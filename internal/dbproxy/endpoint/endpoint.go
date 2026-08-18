// Package endpoint is the kine-style front door for dbproxy: it exposes a
// MySQL-protocol listener (unix socket or TCP) backed by an in-memory
// go-mysql-server engine whose durable state lives in a shared statement log
// (etcd, or an in-process log for single-node non-durable use).
package endpoint

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"strings"
	"sync"

	"github.com/dolthub/go-mysql-server/memory"
	"github.com/dolthub/go-mysql-server/server"
	"github.com/dolthub/go-mysql-server/sql"

	"github.com/metal3-community/metal-boot/internal/dbproxy"
	"github.com/metal3-community/metal-boot/internal/dbproxy/backend"
	"github.com/metal3-community/metal-boot/internal/dbproxy/backend/etcdlog"
	"github.com/metal3-community/metal-boot/internal/dbproxy/backend/memlog"
)

// Config describes a dbproxy endpoint.
type Config struct {
	// Listener is the address to serve the MySQL protocol on, either
	// "unix:///run/metal-boot/db.sock" or "tcp://127.0.0.1:3306".
	Listener string
	// Etcd configures the shared statement log. If Etcd.Endpoints is empty,
	// an in-process, non-durable log is used (single node only).
	Etcd etcdlog.Config
	// Databases to pre-create; defaults to {"ironic"}.
	Databases []string
	// User and Password optionally enable authentication on the MySQL
	// listener. When User is empty, any credentials are accepted.
	User     string
	Password string
}

// Endpoint is a running MySQL-protocol listener backed by the statement log.
type Endpoint struct {
	logger *slog.Logger
	cfg    Config

	network string // "unix" or "tcp"
	addr    string // socket path or host:port actually bound

	log backend.Log
	rep *dbproxy.Replicator
	srv *server.Server

	closeOnce sync.Once
	closeErr  error
	serveDone chan struct{}
}

// Listen builds the statement log, the replicator, and the MySQL server, and
// starts serving. The server shuts down when ctx is canceled or Close is
// called.
func Listen(ctx context.Context, logger *slog.Logger, cfg Config) (*Endpoint, error) {
	if logger == nil {
		logger = slog.Default()
	}
	network, address, err := parseListener(cfg.Listener)
	if err != nil {
		return nil, err
	}
	databases := cfg.Databases
	if len(databases) == 0 {
		databases = []string{"ironic"}
	}

	var log backend.Log
	if len(cfg.Etcd.Endpoints) == 0 {
		logger.Info("dbproxy: no etcd endpoints configured; " +
			"using in-memory statement log (single node, non-durable)")
		log = memlog.New()
	} else {
		log = etcdlog.New(cfg.Etcd)
	}

	rep, err := dbproxy.New(ctx, logger, log, databases)
	if err != nil {
		_ = log.Close()
		return nil, err
	}

	e := &Endpoint{
		logger:    logger,
		cfg:       cfg,
		network:   network,
		log:       log,
		rep:       rep,
		serveDone: make(chan struct{}),
	}

	if cfg.User != "" {
		// Register the configured account as a superuser for both TCP and
		// local socket clients, and run internal replication sessions as it.
		mysqlDb := rep.Engine().Analyzer.Catalog.MySQLDb
		ed := mysqlDb.Editor()
		mysqlDb.AddSuperUser(ed, cfg.User, "%", cfg.Password)
		mysqlDb.AddSuperUser(ed, cfg.User, "localhost", cfg.Password)
		ed.Close()
		rep.SetApplyUser(cfg.User)
	}

	ln, err := net.Listen(network, address)
	if err != nil {
		_ = rep.Close()
		_ = log.Close()
		return nil, fmt.Errorf("dbproxy: listening on %s: %w", cfg.Listener, err)
	}
	e.addr = ln.Addr().String()

	serverCfg := server.Config{
		Protocol: network,
		Address:  e.addr,
		Listener: ln,
		Options:  []server.Option{rep.ServerOption()},
	}
	srv, err := server.NewServer(
		serverCfg,
		rep.Engine(),
		sql.NewContext,
		memory.NewSessionBuilder(rep.Provider()),
		nil,
	)
	if err != nil {
		_ = ln.Close()
		_ = rep.Close()
		_ = log.Close()
		return nil, fmt.Errorf("dbproxy: building MySQL server: %w", err)
	}
	e.srv = srv

	go func() {
		defer close(e.serveDone)
		if err := srv.Start(); err != nil {
			e.logger.Error("dbproxy: MySQL server stopped", "error", err)
		}
	}()
	go func() {
		select {
		case <-ctx.Done():
			_ = e.Close()
		case <-e.serveDone:
		}
	}()

	logger.Info("dbproxy: serving MySQL protocol",
		"network", network, "address", e.addr, "databases", databases,
		"node_id", rep.NodeID())
	return e, nil
}

// Addr returns the bound address: the socket path for unix listeners or
// host:port for TCP listeners.
func (e *Endpoint) Addr() string { return e.addr }

// Replicator returns the endpoint's replicator.
func (e *Endpoint) Replicator() *dbproxy.Replicator { return e.rep }

// DSN returns a SQLAlchemy database URL for the given database, e.g.
// mysql+pymysql://user:pass@localhost/ironic?unix_socket=/run/metal-boot/db.sock
// for unix listeners, or mysql+pymysql://user:pass@127.0.0.1:3306/ironic for
// TCP listeners.
func (e *Endpoint) DSN(database string) string {
	user := e.cfg.User
	if user == "" {
		user = "root"
	}
	cred := url.User(user)
	if e.cfg.Password != "" {
		cred = url.UserPassword(user, e.cfg.Password)
	}
	if e.network == "unix" {
		u := url.URL{
			Scheme:   "mysql+pymysql",
			User:     cred,
			Host:     "localhost",
			Path:     "/" + database,
			RawQuery: url.Values{"unix_socket": {e.addr}}.Encode(),
		}
		return u.String()
	}
	u := url.URL{
		Scheme: "mysql+pymysql",
		User:   cred,
		Host:   e.addr,
		Path:   "/" + database,
	}
	return u.String()
}

// Close shuts down the MySQL server, the replicator, and the statement log.
// It is safe to call multiple times.
func (e *Endpoint) Close() error {
	e.closeOnce.Do(func() {
		var errs []error
		if err := e.srv.Close(); err != nil {
			errs = append(errs, fmt.Errorf("closing MySQL server: %w", err))
		}
		<-e.serveDone
		if err := e.rep.Close(); err != nil {
			errs = append(errs, fmt.Errorf("closing replicator: %w", err))
		}
		if err := e.log.Close(); err != nil {
			errs = append(errs, fmt.Errorf("closing statement log: %w", err))
		}
		if len(errs) > 0 {
			e.closeErr = fmt.Errorf("dbproxy: %w", joinErrors(errs))
		}
	})
	return e.closeErr
}

func joinErrors(errs []error) error {
	if len(errs) == 1 {
		return errs[0]
	}
	msgs := make([]string, len(errs))
	for i, err := range errs {
		msgs[i] = err.Error()
	}
	return fmt.Errorf("%s", strings.Join(msgs, "; "))
}

// parseListener splits a listener URL of the form "unix:///path/to.sock" or
// "tcp://host:port" into a network and address for net.Listen.
func parseListener(listener string) (network, address string, err error) {
	if listener == "" {
		return "", "", fmt.Errorf("dbproxy: listener address is required")
	}
	u, err := url.Parse(listener)
	if err != nil {
		return "", "", fmt.Errorf("dbproxy: parsing listener %q: %w", listener, err)
	}
	switch u.Scheme {
	case "unix":
		path := u.Path
		if u.Host != "" {
			// Tolerate "unix://relative/path" style URLs.
			path = u.Host + u.Path
		}
		if path == "" {
			return "", "", fmt.Errorf("dbproxy: listener %q has no socket path", listener)
		}
		return "unix", path, nil
	case "tcp":
		if u.Host == "" {
			return "", "", fmt.Errorf("dbproxy: listener %q has no host:port", listener)
		}
		return "tcp", u.Host, nil
	default:
		return "", "", fmt.Errorf(
			"dbproxy: unsupported listener scheme %q (want unix:// or tcp://)", u.Scheme)
	}
}
