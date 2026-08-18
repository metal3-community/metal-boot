package dbproxy

import (
	"context"
	"log/slog"
	"strings"

	sqle "github.com/dolthub/go-mysql-server"
	"github.com/dolthub/go-mysql-server/server"
	"github.com/dolthub/vitess/go/mysql"
	"github.com/dolthub/vitess/go/sqltypes"
	querypb "github.com/dolthub/vitess/go/vt/proto/query"
	"github.com/dolthub/vitess/go/vt/sqlparser"
)

// IsWrite reports whether sqlText is a statement that mutates database state
// and therefore must be replicated. Session-scoped statements (SET, USE,
// BEGIN/COMMIT/ROLLBACK) and reads (SELECT — including SELECT ... FOR UPDATE,
// whose locks are per-process — SHOW, EXPLAIN) are not writes.
func IsWrite(sqlText string) (bool, error) {
	stmt, err := sqlparser.Parse(sqlText)
	if err != nil {
		return false, err
	}
	return isWriteStmt(stmt), nil
}

// isWriteStmt classifies a parsed statement. Writes are DML
// (INSERT/REPLACE/UPDATE/DELETE/LOAD DATA) and DDL (CREATE/ALTER/DROP/
// RENAME/TRUNCATE of tables, views and indexes, plus CREATE/DROP DATABASE).
func isWriteStmt(stmt sqlparser.Statement) bool {
	switch stmt.(type) {
	case *sqlparser.Insert, // includes REPLACE
		*sqlparser.Update,
		*sqlparser.Delete,
		*sqlparser.Load,
		*sqlparser.DDL, // CREATE/ALTER/DROP/RENAME/TRUNCATE TABLE, CREATE INDEX, ...
		*sqlparser.AlterTable,
		*sqlparser.DBDDL: // CREATE/DROP DATABASE
		return true
	default:
		return false
	}
}

// ServerOption returns a server.Option that installs a write-replication
// interceptor into the go-mysql-server handler chain. The interceptor lets
// every statement execute normally and, when a write statement succeeds,
// records its SQL text (with the session's current database) to the shared
// statement log.
func (r *Replicator) ServerOption() server.Option {
	return func(
		e *sqle.Engine,
		sm *server.SessionManager,
		h mysql.Handler,
	) (*sqle.Engine, *server.SessionManager, mysql.Handler) {
		var chain server.InterceptorChain
		chain.WithInterceptor(&writeInterceptor{
			logger: r.logger,
			rep:    r,
			sm:     sm,
		})
		return chain.Option()(e, sm, h)
	}
}

// writeInterceptor implements server.Interceptor. It records successful write
// statements to the replicator's log after they execute.
type writeInterceptor struct {
	logger *slog.Logger
	rep    *Replicator
	sm     *server.SessionManager
}

var _ server.Interceptor = (*writeInterceptor)(nil)

// Priority implements server.Interceptor.
func (w *writeInterceptor) Priority() int { return 100 }

// record appends an executed statement to the log. Append failures are
// logged, not returned: the statement already succeeded locally and its OK
// result may already be on the wire, so failing the query would desynchronize
// the client. This leaves a durability gap that is documented in the package
// docs.
func (w *writeInterceptor) record(ctx context.Context, c *mysql.Conn, sqlText string) {
	db := w.sm.GetCurrentDB(c)
	if err := w.rep.Record(ctx, db, sqlText); err != nil {
		w.logger.Error("dbproxy: failed to record write statement; "+
			"local state has diverged from the statement log",
			"db", db, "sql", sqlText, "error", err)
	}
}

// Query implements server.Interceptor.
func (w *writeInterceptor) Query(
	ctx context.Context,
	chain server.Chain,
	c *mysql.Conn,
	query string,
	callback func(res *sqltypes.Result, more bool) error,
) error {
	isWrite, parseErr := IsWrite(query)
	err := chain.ComQuery(ctx, c, query, callback)
	if err == nil && parseErr == nil && isWrite {
		w.record(ctx, c, query)
	}
	return err
}

// ParsedQuery implements server.Interceptor. The handler chain in the current
// go-mysql-server version never invokes it, but the interface requires it.
func (w *writeInterceptor) ParsedQuery(
	chain server.Chain,
	c *mysql.Conn,
	query string,
	parsed sqlparser.Statement,
	callback func(res *sqltypes.Result, more bool) error,
) error {
	ctx := context.Background()
	err := chain.ComQuery(ctx, c, query, callback)
	if err == nil && parsed != nil && isWriteStmt(parsed) {
		w.record(ctx, c, query)
	}
	return err
}

// MultiQuery implements server.Interceptor. Only the executed head of the
// multi-statement string (everything before the returned remainder) is
// classified and recorded.
func (w *writeInterceptor) MultiQuery(
	ctx context.Context,
	chain server.Chain,
	c *mysql.Conn,
	query string,
	callback func(res *sqltypes.Result, more bool) error,
) (string, error) {
	remainder, err := chain.ComMultiQuery(ctx, c, query, callback)
	if err == nil {
		executed := query
		if remainder != "" && strings.HasSuffix(query, remainder) {
			executed = query[:len(query)-len(remainder)]
		}
		executed = strings.TrimSpace(executed)
		if isWrite, parseErr := IsWrite(executed); parseErr == nil && isWrite {
			w.record(ctx, c, executed)
		}
	}
	return remainder, err
}

// Prepare implements server.Interceptor. Preparing a statement has no side
// effects, so it passes through untouched.
func (w *writeInterceptor) Prepare(
	ctx context.Context,
	chain server.Chain,
	c *mysql.Conn,
	query string,
	prepare *mysql.PrepareData,
) ([]*querypb.Field, error) {
	return chain.ComPrepare(ctx, c, query, prepare)
}

// StmtExecute implements server.Interceptor. Binary-protocol executions of
// write statements are recorded by interpolating the bound parameters back
// into the statement text (best effort; failures are logged and the
// statement is not replicated).
func (w *writeInterceptor) StmtExecute(
	ctx context.Context,
	chain server.Chain,
	c *mysql.Conn,
	prepare *mysql.PrepareData,
	callback func(*sqltypes.Result) error,
) error {
	err := chain.ComStmtExecute(ctx, c, prepare, callback)
	if err != nil {
		return err
	}
	if isWrite, parseErr := IsWrite(prepare.PrepareStmt); parseErr == nil && isWrite {
		interpolated, ierr := interpolateBindVars(prepare)
		if ierr != nil {
			w.logger.Warn("dbproxy: cannot replicate prepared write statement",
				"sql", prepare.PrepareStmt, "error", ierr)
			return nil
		}
		w.record(ctx, c, interpolated)
	}
	return nil
}

// interpolateBindVars renders a prepared statement with its bound values
// substituted, producing plain SQL text suitable for the statement log.
func interpolateBindVars(prepare *mysql.PrepareData) (string, error) {
	stmt, err := sqlparser.Parse(prepare.PrepareStmt)
	if err != nil {
		return "", err
	}
	return sqlparser.NewParsedQuery(stmt).GenerateQuery(prepare.BindVars, nil)
}
