package endpoint

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

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

// socketPath returns a path short enough for a unix socket (sun_path is
// limited to ~104 bytes); t.TempDir can exceed that when TMPDIR is deep.
func socketPath(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	sock := filepath.Join(dir, "db.sock")
	if len(sock) <= 90 {
		return sock
	}
	dir, err := os.MkdirTemp("/tmp", "dbproxy")
	if err != nil {
		t.Fatalf("creating short temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return filepath.Join(dir, "db.sock")
}

func openAndPing(t *testing.T, dsn string) *sql.DB {
	t.Helper()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("sql.Open(%q): %v", dsn, err)
	}
	t.Cleanup(func() { _ = db.Close() })
	deadline := time.Now().Add(10 * time.Second)
	for {
		err = db.Ping()
		if err == nil {
			return db
		}
		if time.Now().After(deadline) {
			t.Fatalf("pinging %q: %v", dsn, err)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func TestEndpointUnixSocket(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sock := socketPath(t)
	ep, err := Listen(ctx, testLogger(t), Config{Listener: "unix://" + sock})
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer ep.Close()

	if ep.Addr() != sock {
		t.Errorf("Addr() = %q, want %q", ep.Addr(), sock)
	}

	// interpolateParams keeps go-sql-driver on the text protocol, so the
	// statements land in the log verbatim.
	db := openAndPing(t,
		fmt.Sprintf("root@unix(%s)/ironic?interpolateParams=true", sock))

	const createStmt = "CREATE TABLE nodes (id INT PRIMARY KEY, name VARCHAR(64))"
	const insertStmt = "INSERT INTO nodes VALUES (1, 'node-1')"
	if _, err := db.Exec(createStmt); err != nil {
		t.Fatalf("CREATE TABLE: %v", err)
	}
	if _, err := db.Exec(insertStmt); err != nil {
		t.Fatalf("INSERT: %v", err)
	}

	rows, err := db.Query("SELECT id, name FROM nodes")
	if err != nil {
		t.Fatalf("SELECT: %v", err)
	}
	defer rows.Close()
	var (
		count int
		id    int
		name  string
	)
	for rows.Next() {
		if err := rows.Scan(&id, &name); err != nil {
			t.Fatalf("Scan: %v", err)
		}
		count++
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	if count != 1 || id != 1 || name != "node-1" {
		t.Errorf("SELECT returned count=%d id=%d name=%q, want 1 row (1, node-1)",
			count, id, name)
	}

	// The write statements (and only they) must be in the statement log.
	entries, err := ep.log.List(context.Background(), 0)
	if err != nil {
		t.Fatalf("log.List: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("log has %d entries, want 2: %+v", len(entries), entries)
	}
	if entries[0].SQL != createStmt || entries[1].SQL != insertStmt {
		t.Errorf("log SQL = %q, %q; want %q, %q",
			entries[0].SQL, entries[1].SQL, createStmt, insertStmt)
	}
	for _, e := range entries {
		if e.DB != "ironic" {
			t.Errorf("entry rev %d has DB %q, want %q", e.Rev, e.DB, "ironic")
		}
		if e.NodeID != ep.rep.NodeID() {
			t.Errorf("entry rev %d has NodeID %q, want %q",
				e.Rev, e.NodeID, ep.rep.NodeID())
		}
	}

	dsn := ep.DSN("ironic")
	if !strings.HasPrefix(dsn, "mysql+pymysql://root@localhost/ironic?unix_socket=") {
		t.Errorf("DSN = %q, want mysql+pymysql unix_socket form", dsn)
	}
}

func TestEndpointTCP(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ep, err := Listen(ctx, testLogger(t), Config{Listener: "tcp://127.0.0.1:0"})
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer ep.Close()

	db := openAndPing(t,
		fmt.Sprintf("root@tcp(%s)/ironic?interpolateParams=true", ep.Addr()))
	if _, err := db.Exec("CREATE TABLE t (id INT PRIMARY KEY)"); err != nil {
		t.Fatalf("CREATE TABLE: %v", err)
	}

	dsn := ep.DSN("ironic")
	want := fmt.Sprintf("mysql+pymysql://root@%s/ironic", ep.Addr())
	if dsn != want {
		t.Errorf("DSN = %q, want %q", dsn, want)
	}
}
