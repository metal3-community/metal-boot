package dbproxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"

	sqle "github.com/dolthub/go-mysql-server"
	"github.com/dolthub/go-mysql-server/memory"
	"github.com/dolthub/go-mysql-server/sql"
	"github.com/google/uuid"

	"github.com/metal3-community/metal-boot/internal/dbproxy/backend"
)

// Replicator owns a go-mysql-server engine whose state is rebuilt from, and
// kept in sync with, a shared statement log.
//
// On construction it replays the entire log into a fresh in-memory engine,
// then watches the log and applies entries written by other nodes as they
// arrive. Statements executed locally are applied by the SQL server itself
// and only recorded to the log (see Record); the watcher skips them by
// NodeID.
type Replicator struct {
	logger   *slog.Logger
	log      backend.Log
	engine   *sqle.Engine
	provider *memory.DbProvider
	nodeID   string

	mu          sync.Mutex
	lastApplied int64
	applyUser   string

	cancel context.CancelFunc
	done   chan struct{}
}

// New builds a Replicator: it starts the log, creates an in-memory engine
// with the given databases pre-created, replays the full statement log, and
// starts a watcher that applies statements recorded by other nodes.
//
// Entries that fail to replay are logged at warn level and skipped so a
// single bad statement cannot prevent startup.
func New(
	ctx context.Context,
	logger *slog.Logger,
	log backend.Log,
	dbNames []string,
) (*Replicator, error) {
	if logger == nil {
		logger = slog.Default()
	}
	if err := log.Start(ctx); err != nil {
		return nil, fmt.Errorf("dbproxy: starting statement log: %w", err)
	}

	dbs := make([]sql.Database, 0, len(dbNames))
	for _, name := range dbNames {
		db := memory.NewDatabase(name)
		db.BaseDatabase.EnablePrimaryKeyIndexes()
		dbs = append(dbs, db)
	}
	provider := memory.NewDBProvider(dbs...)

	r := &Replicator{
		logger:    logger,
		log:       log,
		engine:    sqle.NewDefault(provider),
		provider:  provider,
		nodeID:    uuid.NewString(),
		applyUser: "root",
		done:      make(chan struct{}),
	}

	// Replay the full log to rebuild state.
	entries, err := log.List(ctx, 0)
	if err != nil {
		return nil, fmt.Errorf("dbproxy: listing statement log: %w", err)
	}
	for _, e := range entries {
		if err := r.apply(ctx, e); err != nil {
			logger.Warn("dbproxy: skipping log entry that failed to replay",
				"rev", e.Rev, "db", e.DB, "sql", e.SQL, "error", err)
		}
		r.lastApplied = e.Rev
	}
	logger.Info("dbproxy: statement log replayed",
		"entries", len(entries), "last_applied", r.lastApplied, "node_id", r.nodeID)

	// Follow the log for statements from other nodes. The watcher must
	// outlive the (possibly short-lived) construction context.
	wctx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	r.cancel = cancel
	ch, err := log.Watch(wctx, r.lastApplied)
	if err != nil {
		cancel()
		close(r.done)
		return nil, fmt.Errorf("dbproxy: watching statement log: %w", err)
	}
	go r.consume(wctx, ch)
	return r, nil
}

// Engine returns the underlying go-mysql-server engine.
func (r *Replicator) Engine() *sqle.Engine { return r.engine }

// Provider returns the in-memory database provider backing the engine.
func (r *Replicator) Provider() *memory.DbProvider { return r.provider }

// NodeID returns this process's random node identity.
func (r *Replicator) NodeID() string { return r.nodeID }

// LastApplied returns the highest log revision known to be reflected in the
// local engine state.
func (r *Replicator) LastApplied() int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.lastApplied
}

// SetApplyUser sets the client user internal apply sessions run as. It only
// matters when user accounts are enabled on the engine (the user must exist
// and hold sufficient privileges); the default is "root".
func (r *Replicator) SetApplyUser(user string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.applyUser = user
}

// Record appends a statement that was already executed successfully by the
// local engine to the shared log. It is safe for concurrent use.
func (r *Replicator) Record(ctx context.Context, db, sqlText string) error {
	rev, err := r.log.Append(ctx, backend.Entry{
		NodeID: r.nodeID,
		DB:     db,
		SQL:    sqlText,
	})
	if err != nil {
		return fmt.Errorf("dbproxy: appending statement to log: %w", err)
	}
	r.mu.Lock()
	if rev > r.lastApplied {
		r.lastApplied = rev
	}
	r.mu.Unlock()
	return nil
}

// Exec executes a statement against the local engine without recording it to
// the log. It is useful for local bootstrap and tests.
func (r *Replicator) Exec(ctx context.Context, db, sqlText string) error {
	return r.apply(ctx, backend.Entry{DB: db, SQL: sqlText})
}

// Close stops the log watcher and shuts the engine down. It does not close
// the underlying Log; the caller that supplied it owns it.
func (r *Replicator) Close() error {
	r.cancel()
	<-r.done
	// The engine reports its own background-thread cancellation as
	// context.Canceled; that is the expected shutdown path, not an error.
	if err := r.engine.Close(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

// consume applies entries streamed from the log watch. Entries recorded by
// this node were already applied at execution time and are skipped.
func (r *Replicator) consume(ctx context.Context, ch <-chan backend.Entry) {
	defer close(r.done)
	lastSeen := r.LastApplied()
	for {
		select {
		case <-ctx.Done():
			return
		case e, ok := <-ch:
			if !ok {
				if ctx.Err() == nil {
					r.logger.Warn("dbproxy: statement log watch closed unexpectedly")
				}
				return
			}
			if e.Rev <= lastSeen {
				continue
			}
			lastSeen = e.Rev
			if e.NodeID != r.nodeID {
				if err := r.apply(ctx, e); err != nil {
					r.logger.Warn("dbproxy: failed to apply replicated statement",
						"rev", e.Rev, "node_id", e.NodeID, "db", e.DB, "sql", e.SQL,
						"error", err)
				} else {
					r.logger.Debug("dbproxy: applied replicated statement",
						"rev", e.Rev, "node_id", e.NodeID, "db", e.DB)
				}
			}
			r.mu.Lock()
			if e.Rev > r.lastApplied {
				r.lastApplied = e.Rev
			}
			r.mu.Unlock()
		}
	}
}

// apply runs one statement on the local engine in a fresh internal session
// bound to the entry's database, draining the result iterator.
func (r *Replicator) apply(ctx context.Context, e backend.Entry) error {
	sqlCtx := r.newContext(ctx)
	if e.DB != "" {
		sqlCtx.SetCurrentDatabase(e.DB)
	}
	_, iter, _, err := r.engine.Query(sqlCtx, e.SQL)
	if err != nil {
		return err
	}
	for {
		if _, err := iter.Next(sqlCtx); err != nil {
			closeErr := iter.Close(sqlCtx)
			if errors.Is(err, io.EOF) {
				return closeErr
			}
			return err
		}
	}
}

// newContext builds a sql.Context with a fresh internal session against the
// memory provider.
func (r *Replicator) newContext(ctx context.Context) *sql.Context {
	base := sql.NewBaseSession()
	r.mu.Lock()
	user := r.applyUser
	r.mu.Unlock()
	base.SetClient(sql.Client{User: user, Address: "localhost"})
	session := memory.NewSession(base, r.provider)
	return sql.NewContext(ctx, sql.WithSession(session))
}
