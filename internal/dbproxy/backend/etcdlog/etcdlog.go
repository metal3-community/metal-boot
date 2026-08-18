// Package etcdlog implements backend.Log on top of etcd, using the
// go.etcd.io/etcd/client/v3 client.
//
// Layout under the configured prefix (default "/metal-boot/dbproxy"):
//
//	<prefix>/seq              the current (latest) log revision, as decimal text
//	<prefix>/log/<%020d rev>  one JSON-encoded backend.Entry per revision
//
// Appends allocate revisions with a compare-and-swap transaction on the seq
// key, so revisions are dense and strictly ordered even with concurrent
// writers. Because log keys are zero-padded, etcd's key order equals revision
// order, which List and Watch rely on.
package etcdlog

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/metal3-community/metal-boot/internal/dbproxy/backend"
)

// DefaultPrefix is the default etcd key prefix for the statement log.
const DefaultPrefix = "/metal-boot/dbproxy"

// DefaultDialTimeout is used when Config.DialTimeout is zero.
const DefaultDialTimeout = 5 * time.Second

// ErrNotStarted is returned when the log is used before Start succeeds.
var ErrNotStarted = errors.New("etcdlog: log not started")

// Config describes the etcd connection and key layout.
type Config struct {
	// Endpoints lists etcd server URLs, e.g. ["https://127.0.0.1:2379"].
	Endpoints []string
	// Prefix is the etcd key prefix; DefaultPrefix if empty.
	Prefix string
	// CertFile and KeyFile optionally configure a client certificate.
	CertFile string
	KeyFile  string
	// CAFile optionally configures the CA bundle used to verify the server.
	CAFile string
	// DialTimeout bounds the initial dial; DefaultDialTimeout if zero.
	DialTimeout time.Duration
}

// Log is an etcd-backed backend.Log.
type Log struct {
	cfg    Config
	client *clientv3.Client
}

var _ backend.Log = (*Log)(nil)

// New returns an unstarted etcd log. Call Start before use.
func New(cfg Config) *Log {
	if cfg.Prefix == "" {
		cfg.Prefix = DefaultPrefix
	}
	if cfg.DialTimeout == 0 {
		cfg.DialTimeout = DefaultDialTimeout
	}
	return &Log{cfg: cfg}
}

func (l *Log) seqKey() string    { return l.cfg.Prefix + "/seq" }
func (l *Log) logPrefix() string { return l.cfg.Prefix + "/log/" }

func (l *Log) key(rev int64) string {
	return fmt.Sprintf("%s%020d", l.logPrefix(), rev)
}

// Start implements backend.Log. It dials etcd; it is idempotent.
func (l *Log) Start(_ context.Context) error {
	if l.client != nil {
		return nil
	}
	clientCfg := clientv3.Config{
		Endpoints:   l.cfg.Endpoints,
		DialTimeout: l.cfg.DialTimeout,
	}
	if l.cfg.CertFile != "" || l.cfg.KeyFile != "" || l.cfg.CAFile != "" {
		tlsCfg, err := l.tlsConfig()
		if err != nil {
			return err
		}
		clientCfg.TLS = tlsCfg
	}
	client, err := clientv3.New(clientCfg)
	if err != nil {
		return fmt.Errorf("etcdlog: dialing etcd: %w", err)
	}
	l.client = client
	return nil
}

func (l *Log) tlsConfig() (*tls.Config, error) {
	tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12}
	if l.cfg.CertFile != "" || l.cfg.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(l.cfg.CertFile, l.cfg.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("etcdlog: loading client certificate: %w", err)
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}
	if l.cfg.CAFile != "" {
		pem, err := os.ReadFile(l.cfg.CAFile)
		if err != nil {
			return nil, fmt.Errorf("etcdlog: reading CA file: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("etcdlog: no certificates found in %s", l.cfg.CAFile)
		}
		tlsCfg.RootCAs = pool
	}
	return tlsCfg, nil
}

// Append implements backend.Log. Revisions are allocated with a CAS loop on
// the seq key; the seq update and the log-entry put commit atomically.
func (l *Log) Append(ctx context.Context, e backend.Entry) (int64, error) {
	if l.client == nil {
		return 0, ErrNotStarted
	}
	for {
		resp, err := l.client.Get(ctx, l.seqKey())
		if err != nil {
			return 0, fmt.Errorf("etcdlog: reading seq: %w", err)
		}
		var cur int64
		var cmp clientv3.Cmp
		if len(resp.Kvs) == 0 {
			cmp = clientv3.Compare(clientv3.CreateRevision(l.seqKey()), "=", 0)
		} else {
			cur, err = strconv.ParseInt(string(resp.Kvs[0].Value), 10, 64)
			if err != nil {
				return 0, fmt.Errorf("etcdlog: corrupt seq key %q: %w", resp.Kvs[0].Value, err)
			}
			cmp = clientv3.Compare(clientv3.ModRevision(l.seqKey()), "=", resp.Kvs[0].ModRevision)
		}

		next := cur + 1
		e.Rev = next
		data, err := json.Marshal(e)
		if err != nil {
			return 0, fmt.Errorf("etcdlog: encoding entry: %w", err)
		}

		txn, err := l.client.Txn(ctx).If(cmp).Then(
			clientv3.OpPut(l.seqKey(), strconv.FormatInt(next, 10)),
			clientv3.OpPut(l.key(next), string(data)),
		).Commit()
		if err != nil {
			return 0, fmt.Errorf("etcdlog: committing entry: %w", err)
		}
		if txn.Succeeded {
			return next, nil
		}
		// Lost the CAS race; retry unless the context is done.
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}
	}
}

// List implements backend.Log.
func (l *Log) List(ctx context.Context, sinceRev int64) ([]backend.Entry, error) {
	entries, _, err := l.list(ctx, sinceRev)
	return entries, err
}

// list returns entries with Rev > sinceRev plus the etcd store revision the
// read was served at, which Watch uses as its starting point.
func (l *Log) list(ctx context.Context, sinceRev int64) ([]backend.Entry, int64, error) {
	if l.client == nil {
		return nil, 0, ErrNotStarted
	}
	if sinceRev < 0 {
		sinceRev = 0
	}
	// Start from the key of sinceRev+1 rather than scanning the whole log.
	start := l.key(sinceRev + 1)
	end := clientv3.GetPrefixRangeEnd(l.logPrefix())
	resp, err := l.client.Get(ctx, start,
		clientv3.WithRange(end),
		clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("etcdlog: listing entries: %w", err)
	}
	entries := make([]backend.Entry, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		var e backend.Entry
		if err := json.Unmarshal(kv.Value, &e); err != nil {
			return nil, 0, fmt.Errorf("etcdlog: corrupt entry at %s: %w", kv.Key, err)
		}
		if e.Rev <= sinceRev {
			continue
		}
		entries = append(entries, e)
	}
	return entries, resp.Header.Revision, nil
}

// Watch implements backend.Log. It lists existing entries first, then follows
// new appends with an etcd watch started at the store revision of the list,
// so no entry is missed or delivered twice.
func (l *Log) Watch(ctx context.Context, sinceRev int64) (<-chan backend.Entry, error) {
	if l.client == nil {
		return nil, ErrNotStarted
	}
	initial, storeRev, err := l.list(ctx, sinceRev)
	if err != nil {
		return nil, err
	}
	wch := l.client.Watch(ctx, l.logPrefix(),
		clientv3.WithPrefix(),
		clientv3.WithRev(storeRev+1),
	)
	out := make(chan backend.Entry, 64)
	go func() {
		defer close(out)
		last := sinceRev
		for _, e := range initial {
			select {
			case out <- e:
				last = e.Rev
			case <-ctx.Done():
				return
			}
		}
		for wresp := range wch {
			if wresp.Err() != nil {
				return
			}
			for _, ev := range wresp.Events {
				if ev.Type != clientv3.EventTypePut {
					continue
				}
				var e backend.Entry
				if err := json.Unmarshal(ev.Kv.Value, &e); err != nil {
					continue
				}
				if e.Rev <= last {
					continue
				}
				select {
				case out <- e:
					last = e.Rev
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out, nil
}

// CurrentRev implements backend.Log.
func (l *Log) CurrentRev(ctx context.Context) (int64, error) {
	if l.client == nil {
		return 0, ErrNotStarted
	}
	resp, err := l.client.Get(ctx, l.seqKey())
	if err != nil {
		return 0, fmt.Errorf("etcdlog: reading seq: %w", err)
	}
	if len(resp.Kvs) == 0 {
		return 0, nil
	}
	rev, err := strconv.ParseInt(string(resp.Kvs[0].Value), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("etcdlog: corrupt seq key %q: %w", resp.Kvs[0].Value, err)
	}
	return rev, nil
}

// Close implements backend.Log.
func (l *Log) Close() error {
	if l.client == nil {
		return nil
	}
	err := l.client.Close()
	l.client = nil
	return err
}
