package etcdlog

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/metal3-community/metal-boot/internal/dbproxy/backend"
)

// TestEtcdLogRoundTrip exercises Append/List/Watch/CurrentRev against a real
// etcd. It is skipped unless DBPROXY_TEST_ETCD_ENDPOINTS is set, e.g.
//
//	DBPROXY_TEST_ETCD_ENDPOINTS=http://127.0.0.1:2379 go test ./...
func TestEtcdLogRoundTrip(t *testing.T) {
	endpoints := os.Getenv("DBPROXY_TEST_ETCD_ENDPOINTS")
	if endpoints == "" {
		t.Skip("DBPROXY_TEST_ETCD_ENDPOINTS not set; skipping etcd integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log := New(Config{
		Endpoints: strings.Split(endpoints, ","),
		Prefix:    fmt.Sprintf("/metal-boot/dbproxy-test/%d", time.Now().UnixNano()),
	})
	if err := log.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer log.Close()

	if rev, err := log.CurrentRev(ctx); err != nil || rev != 0 {
		t.Fatalf("CurrentRev on empty log = %d, %v; want 0, nil", rev, err)
	}

	want := []backend.Entry{
		{Rev: 1, NodeID: "n1", DB: "ironic", SQL: "CREATE TABLE t (id INT PRIMARY KEY)"},
		{Rev: 2, NodeID: "n1", DB: "ironic", SQL: "INSERT INTO t VALUES (1)"},
		{Rev: 3, NodeID: "n2", DB: "ironic", SQL: "INSERT INTO t VALUES (2)"},
	}
	for i, e := range want {
		rev, err := log.Append(ctx, backend.Entry{
			NodeID: e.NodeID, DB: e.DB, SQL: e.SQL,
		})
		if err != nil {
			t.Fatalf("Append(%d): %v", i, err)
		}
		if rev != e.Rev {
			t.Fatalf("Append(%d) assigned rev %d, want %d", i, rev, e.Rev)
		}
	}

	if rev, err := log.CurrentRev(ctx); err != nil || rev != 3 {
		t.Fatalf("CurrentRev = %d, %v; want 3, nil", rev, err)
	}

	got, err := log.List(ctx, 0)
	if err != nil {
		t.Fatalf("List(0): %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("List(0) = %+v, want %+v", got, want)
	}

	got, err = log.List(ctx, 2)
	if err != nil {
		t.Fatalf("List(2): %v", err)
	}
	if !reflect.DeepEqual(got, want[2:]) {
		t.Errorf("List(2) = %+v, want %+v", got, want[2:])
	}

	// Watch must deliver existing entries after sinceRev, then live appends.
	wctx, wcancel := context.WithCancel(ctx)
	defer wcancel()
	ch, err := log.Watch(wctx, 1)
	if err != nil {
		t.Fatalf("Watch(1): %v", err)
	}
	recv := func() backend.Entry {
		t.Helper()
		select {
		case e, ok := <-ch:
			if !ok {
				t.Fatal("watch channel closed early")
			}
			return e
		case <-time.After(10 * time.Second):
			t.Fatal("timed out waiting for watch entry")
			return backend.Entry{}
		}
	}
	if e := recv(); !reflect.DeepEqual(e, want[1]) {
		t.Errorf("watch entry = %+v, want %+v", e, want[1])
	}
	if e := recv(); !reflect.DeepEqual(e, want[2]) {
		t.Errorf("watch entry = %+v, want %+v", e, want[2])
	}

	live := backend.Entry{Rev: 4, NodeID: "n2", DB: "ironic", SQL: "DELETE FROM t WHERE id = 1"}
	if _, err := log.Append(ctx, backend.Entry{
		NodeID: live.NodeID, DB: live.DB, SQL: live.SQL,
	}); err != nil {
		t.Fatalf("Append(live): %v", err)
	}
	if e := recv(); !reflect.DeepEqual(e, live) {
		t.Errorf("live watch entry = %+v, want %+v", e, live)
	}
}
