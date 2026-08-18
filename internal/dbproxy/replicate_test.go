package dbproxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/metal3-community/metal-boot/internal/dbproxy/backend/memlog"
)

const testDB = "testdb"

func testLogger(t *testing.T) *slog.Logger {
	t.Helper()
	return slog.New(slog.NewTextHandler(
		testWriter{t}, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) {
	w.t.Log(strings.TrimSuffix(string(p), "\n"))
	return len(p), nil
}

// execAndRecord mimics the interceptor: execute locally, then append to the
// shared log.
func execAndRecord(t *testing.T, r *Replicator, db, query string) {
	t.Helper()
	ctx := context.Background()
	if err := r.Exec(ctx, db, query); err != nil {
		t.Fatalf("Exec(%q): %v", query, err)
	}
	if err := r.Record(ctx, db, query); err != nil {
		t.Fatalf("Record(%q): %v", query, err)
	}
}

// tryQuery runs a read query on the replicator's engine, returning the rows
// with every column rendered via fmt.Sprint.
func tryQuery(r *Replicator, db, query string) ([][]string, error) {
	sqlCtx := r.newContext(context.Background())
	sqlCtx.SetCurrentDatabase(db)
	_, iter, _, err := r.engine.Query(sqlCtx, query)
	if err != nil {
		return nil, err
	}
	defer func() { _ = iter.Close(sqlCtx) }()
	var rows [][]string
	for {
		row, err := iter.Next(sqlCtx)
		if errors.Is(err, io.EOF) {
			return rows, nil
		}
		if err != nil {
			return nil, err
		}
		cols := make([]string, len(row))
		for i, v := range row {
			cols[i] = fmt.Sprint(v)
		}
		rows = append(rows, cols)
	}
}

func mustQuery(t *testing.T, r *Replicator, db, query string) [][]string {
	t.Helper()
	rows, err := tryQuery(r, db, query)
	if err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	return rows
}

func TestReplicatorReplayOnStartup(t *testing.T) {
	ctx := context.Background()
	log := memlog.New()
	defer log.Close()

	a, err := New(ctx, testLogger(t), log, []string{testDB})
	if err != nil {
		t.Fatalf("New(a): %v", err)
	}
	defer a.Close()

	execAndRecord(t, a, testDB,
		"CREATE TABLE t1 (id INT PRIMARY KEY, name VARCHAR(32))")
	execAndRecord(t, a, testDB, "INSERT INTO t1 VALUES (1, 'one')")
	execAndRecord(t, a, testDB, "INSERT INTO t1 VALUES (2, 'two')")

	// A brand-new replicator on the same log must rebuild the table and rows
	// during startup replay.
	b, err := New(ctx, testLogger(t), log, []string{testDB})
	if err != nil {
		t.Fatalf("New(b): %v", err)
	}
	defer b.Close()

	got := mustQuery(t, b, testDB, "SELECT id, name FROM t1 ORDER BY id")
	want := [][]string{{"1", "one"}, {"2", "two"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("replayed rows = %v, want %v", got, want)
	}
	if b.LastApplied() != 3 {
		t.Errorf("b.LastApplied() = %d, want 3", b.LastApplied())
	}
}

func TestReplicatorLiveWatch(t *testing.T) {
	ctx := context.Background()
	log := memlog.New()
	defer log.Close()

	a, err := New(ctx, testLogger(t), log, []string{testDB})
	if err != nil {
		t.Fatalf("New(a): %v", err)
	}
	defer a.Close()

	b, err := New(ctx, testLogger(t), log, []string{testDB})
	if err != nil {
		t.Fatalf("New(b): %v", err)
	}
	defer b.Close()

	execAndRecord(t, a, testDB,
		"CREATE TABLE live (id INT PRIMARY KEY, v VARCHAR(16))")
	execAndRecord(t, a, testDB, "INSERT INTO live VALUES (7, 'seven')")

	// B applies A's statements via its live watch on the shared log.
	want := [][]string{{"7", "seven"}}
	deadline := time.Now().Add(10 * time.Second)
	for {
		got, err := tryQuery(b, testDB, "SELECT id, v FROM live ORDER BY id")
		if err == nil && reflect.DeepEqual(got, want) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("b never saw replicated rows: rows=%v err=%v", got, err)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestReplicatorRestartDurability(t *testing.T) {
	ctx := context.Background()
	log := memlog.New()
	defer log.Close()

	a, err := New(ctx, testLogger(t), log, []string{testDB})
	if err != nil {
		t.Fatalf("New(a): %v", err)
	}
	execAndRecord(t, a, testDB, "CREATE TABLE d (id INT PRIMARY KEY)")
	execAndRecord(t, a, testDB, "INSERT INTO d VALUES (42)")
	if err := a.Close(); err != nil {
		t.Fatalf("a.Close(): %v", err)
	}

	// A fresh replicator on the same (still open) log sees the data: the log,
	// not the engine, is the durable state.
	c, err := New(ctx, testLogger(t), log, []string{testDB})
	if err != nil {
		t.Fatalf("New(c): %v", err)
	}
	defer c.Close()

	got := mustQuery(t, c, testDB, "SELECT id FROM d")
	want := [][]string{{"42"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("rows after restart = %v, want %v", got, want)
	}
}
