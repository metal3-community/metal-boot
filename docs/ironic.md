# Serving OpenStack Ironic from metal-boot

metal-boot can host a complete standalone Ironic without MariaDB, without a
separate WSGI server, and with multiple replicas serving DHCP. Three
subsystems cooperate:

## 1. Embedded WSGI API (`ironic.embedded_wsgi`)

Instead of supervising the all-in-one `ironic` process and reverse-proxying
`/v1/` to its unix socket, metal-boot can embed a CPython interpreter (cgo)
and call the Ironic WSGI application directly from its own HTTP mux,
following the architecture of [wsgo](https://github.com/jonny5532/wsgo):

- one interpreter per process, initialized on a locked OS thread;
- a pool of worker goroutines, each pinned to an OS thread with a persistent
  `PyThreadState`, executing `application(environ, start_response)`;
- the GIL released around request-body reads and response-body writes;
- request bodies buffered *before* a Python worker is occupied;
- stuck requests interrupted with an asynchronous `TimeoutError`
  (`PyThreadState_SetAsyncExc`) and leaked `logging` locks released.

The conductor still runs as a supervised subprocess (`ironic-conductor`),
talking to the embedded API over `json-rpc` (`rpc_transport` is switched
automatically when `embedded_wsgi` is enabled).

Build requirements: the `ironic_wsgi` build tag and the `python3-embed`
pkg-config package (plus Ironic importable by that Python):

```sh
go build -tags ironic_wsgi ./cmd/metal-boot
```

Without the tag the binary still builds; enabling `embedded_wsgi` then logs
an error and falls back to the socket proxy.

```yaml
ironic:
  enabled: true
  supervisor_enabled: true
  embedded_wsgi: true
  wsgi_workers: 4          # Python worker threads
  python_path: []          # extra sys.path entries
```

## 2. etcd database proxy (`ironic.db_proxy`)

A kine-style shim (mirror image of [kine](https://github.com/k3s-io/kine):
kine serves the etcd API from SQL; this serves the MySQL protocol from
etcd). metal-boot listens on a MySQL-protocol unix socket/TCP port using an
embedded SQL engine; every successfully executed write statement is appended
to an ordered statement log in etcd, replayed on startup and streamed to
other replicas by watch. Ironic connects with plain
`mysql+pymysql://` and needs no MariaDB.

```yaml
ironic:
  db_proxy:
    enabled: true
    listener: unix:///tmp/metal-boot-db.sock
    etcd_endpoints: ["https://etcd.kube-system.svc:2379"]
    etcd_prefix: /metal-boot/dbproxy
```

When enabled, `ironic.database_connection` is overridden automatically with
the proxy's DSN. With no etcd endpoints configured the proxy runs on an
in-memory log (single node, non-durable) — useful for development.

Limitations (documented in `internal/dbproxy`): statement-based replication
is last-write-wins and transactions are not replicated atomically; deploy
single-writer (leader-elected) replicas for production.

## 3. Replica-friendly DHCP (`dhcp.macvlan_enabled`)

Ported from Tinkerbell smee's interface hack: each replica creates a
bridge-mode macvlan on the host uplink **in the host network namespace**
(entered via `/proc/1/ns/net`), moves it into its own pod namespace, and
binds the DHCP listener to it (`SO_BINDTODEVICE`; `SO_REUSEPORT` comes from
the DHCP socket helper). The interface is named `mv<first 13 hex of pod
UID>` so replicas never collide, carries `169.254.1.1/32` with
`IFA_F_NOPREFIXROUTE`, and gets an explicit `255.255.255.255/32`
`RTN_BROADCAST` route in the local table (without which broadcast replies
fail with `EINVAL`). Stale `mv*` interfaces found in the host namespace are
purged at startup, and the interface is torn down on shutdown.

Pod requirements: `CAP_NET_ADMIN`, `CAP_SYS_ADMIN`, `hostPID: true`, and the
downward-API env var:

```yaml
env:
  - name: MY_POD_UID
    valueFrom: {fieldRef: {fieldPath: metadata.uid}}
```

```yaml
dhcp:
  enabled: true
  proxy_enabled: true
  macvlan_enabled: true
  macvlan_source_interface: ""   # empty = host default-route interface
```

Without macvlan, metal-boot still binds the DHCP socket to the default-route
interface when `dhcp.interface` is unset, so broadcast replies have a valid
source on multi-homed/CNI pods.
