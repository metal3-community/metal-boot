// Package memlog provides an in-memory, in-process implementation of
// backend.Log. It is non-durable and is intended for tests and for
// single-node deployments that do not require persistence.
package memlog

import (
	"context"
	"errors"

	"sync"

	"github.com/metal3-community/metal-boot/internal/dbproxy/backend"
)

// ErrClosed is returned by Append after the log has been closed.
var ErrClosed = errors.New("memlog: log is closed")

// Log is an in-memory backend.Log. The zero value is not usable; use New.
type Log struct {
	mu      sync.Mutex
	entries []backend.Entry
	// notify is closed and replaced on every append so watchers can wait
	// for new entries without polling.
	notify chan struct{}
	done   chan struct{}
	closed bool
}

var _ backend.Log = (*Log)(nil)

// New returns an empty in-memory log.
func New() *Log {
	return &Log{
		notify: make(chan struct{}),
		done:   make(chan struct{}),
	}
}

// Start implements backend.Log. It is a no-op for the in-memory log.
func (l *Log) Start(context.Context) error { return nil }

// Append implements backend.Log.
func (l *Log) Append(_ context.Context, e backend.Entry) (int64, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return 0, ErrClosed
	}
	e.Rev = int64(len(l.entries)) + 1
	l.entries = append(l.entries, e)
	close(l.notify)
	l.notify = make(chan struct{})
	return e.Rev, nil
}

// List implements backend.Log.
func (l *Log) List(_ context.Context, sinceRev int64) ([]backend.Entry, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if sinceRev < 0 {
		sinceRev = 0
	}
	if sinceRev >= int64(len(l.entries)) {
		return nil, nil
	}
	tail := l.entries[sinceRev:]
	out := make([]backend.Entry, len(tail))
	copy(out, tail)
	return out, nil
}

// Watch implements backend.Log.
func (l *Log) Watch(ctx context.Context, sinceRev int64) (<-chan backend.Entry, error) {
	out := make(chan backend.Entry)
	go func() {
		defer close(out)
		next := sinceRev + 1
		if next < 1 {
			next = 1
		}
		for {
			l.mu.Lock()
			var e backend.Entry
			have := next <= int64(len(l.entries))
			if have {
				e = l.entries[next-1]
			}
			wait := l.notify
			l.mu.Unlock()

			if have {
				select {
				case out <- e:
					next++
				case <-ctx.Done():
					return
				case <-l.done:
					return
				}
				continue
			}

			select {
			case <-wait:
			case <-ctx.Done():
				return
			case <-l.done:
				return
			}
		}
	}()
	return out, nil
}

// CurrentRev implements backend.Log.
func (l *Log) CurrentRev(context.Context) (int64, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return int64(len(l.entries)), nil
}

// Close implements backend.Log.
func (l *Log) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.closed {
		l.closed = true
		close(l.done)
	}
	return nil
}
