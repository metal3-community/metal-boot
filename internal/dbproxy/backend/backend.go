// Package backend defines the replicated statement-log abstraction used by
// the dbproxy package.
//
// A Log is an ordered, append-only sequence of SQL statements. Revisions are
// assigned by the log at append time, start at 1, and are strictly
// monotonically increasing with no gaps. Implementations must deliver entries
// from Watch in ascending revision order and exactly once per watcher.
package backend

import "context"

// Entry is a single logged SQL statement.
type Entry struct {
	// Rev is the revision assigned by the log when the entry was appended.
	Rev int64 `json:"rev"`
	// NodeID identifies the process that executed and appended the statement.
	NodeID string `json:"node_id"`
	// DB is the session's current database when the statement executed
	// (may be empty, e.g. for CREATE DATABASE).
	DB string `json:"db"`
	// SQL is the statement text.
	SQL string `json:"sql"`
}

// Log is an ordered, shared statement log.
type Log interface {
	// Start initializes the log (e.g. dials etcd). It must be called before
	// any other method and is idempotent.
	Start(ctx context.Context) error
	// Append appends a statement, returning its assigned revision.
	Append(ctx context.Context, e Entry) (int64, error)
	// List returns entries with Rev > sinceRev in ascending order.
	List(ctx context.Context, sinceRev int64) ([]Entry, error)
	// Watch streams entries with Rev > sinceRev, following new appends until
	// ctx ends. The returned channel is closed when the watch terminates.
	Watch(ctx context.Context, sinceRev int64) (<-chan Entry, error)
	// CurrentRev returns the revision of the newest entry (0 if empty).
	CurrentRev(ctx context.Context) (int64, error)
	// Close releases resources held by the log. Watches are terminated.
	Close() error
}
