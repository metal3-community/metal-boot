// Package dbproxy implements a MySQL-protocol database endpoint for
// OpenStack Ironic whose durable state lives in Kubernetes' etcd.
//
// # Design
//
// The design is modeled on k3s-io/kine, mirrored: kine exposes an etcd
// endpoint backed by a SQL database; dbproxy exposes a SQL (MySQL wire
// protocol) endpoint backed by etcd. Ironic (SQLAlchemy + PyMySQL) connects
// to this endpoint over a unix socket or localhost TCP instead of requiring
// MariaDB.
//
// SQL execution is handled by github.com/dolthub/go-mysql-server (GMS) with
// its in-memory storage provider. Durability and multi-replica replication
// come from a statement log stored in etcd (see backend, backend/etcdlog and
// backend/memlog):
//
//   - Every write statement (DML and DDL) that executes successfully is
//     appended to an ordered log in etcd as a backend.Entry
//     {Rev, NodeID, DB, SQL}.
//   - On startup, a Replicator replays the whole log against a fresh
//     in-memory engine to rebuild state. Entries that fail to replay are
//     logged at warn level and skipped so a bad statement cannot prevent
//     startup.
//   - A watch on the log applies statements appended by other replicas live.
//     Each process has a random NodeID (a UUID) and skips its own entries,
//     which were already applied locally at execution time.
//
// The front door is endpoint.Listen, which wires a statement log, a
// Replicator, and a GMS server together and serves the MySQL protocol on a
// unix socket or TCP address.
//
// # Write interception
//
// Statements are intercepted with GMS's server.Interceptor /
// server.InterceptorChain API (installed via server.Config.Options through
// Replicator.ServerOption). This is GMS's supported hook for wrapping the
// wire-protocol handler: the interceptor lets each statement execute
// normally, and when a write statement (classified by IsWrite using the
// github.com/dolthub/vitess/go/vt/sqlparser dialect) succeeds it records the
// statement text and the session's current database to the log.
//
// Text-protocol queries (used by PyMySQL, Ironic's driver) are recorded
// verbatim. Binary-protocol prepared statements (COM_STMT_EXECUTE) are
// recorded by interpolating the bound parameters back into the statement
// text, best effort.
//
// # Limitations
//
//   - Replication is last-write-wins at statement granularity. There is no
//     cross-replica locking or conflict detection; concurrent writers to the
//     same rows on different replicas can diverge. Deploy a single writer
//     (e.g. via leader election) for now.
//   - Transactions execute locally; only individual successful write
//     statements are replicated. A rolled-back transaction replicates
//     nothing, but atomicity of a multi-statement transaction is not
//     preserved across replicas: each write statement is shipped
//     independently as it commits locally.
//   - SET statements are session-scoped and are not replicated; SELECT ...
//     FOR UPDATE executes locally and its locks are per-process.
//   - If appending to the log fails after a statement has executed locally,
//     the statement stays applied locally but is not replicated (an error is
//     logged). The OK packet may already be on the wire, so the client is
//     not failed retroactively.
//   - Account management (CREATE USER, GRANT, ...) is not replicated.
package dbproxy
