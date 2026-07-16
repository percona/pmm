# Proof: sharing PMM's built-in PostgreSQL with a side container

Two Docker Compose files, identical topology — **two separate containers on a shared
user-defined bridge network** (separate network namespaces; no `network_mode: "container:"`).
The only difference is whether PMM's PG is opened up. The same probe tool runs in both.

The variable being isolated is `listen_addresses` (in `postgresql.conf`), which is what the
"listen on all interfaces" question is actually about — NOT `pg_hba.conf`.

## Fail path — `pg-share-fail.compose.yml`
PMM's PG left untouched → binds `127.0.0.1` only. The client hits pmm-server's `eth0`, where
nothing listens → connection refused.

```
docker compose -f pg-share-fail.compose.yml up --abort-on-container-exit --exit-code-from client
docker compose -f pg-share-fail.compose.yml down -v   # cleanup
```
Expected client output: `pg_isready` → `no response` (rc=2); `psql` → `Connection refused`.

## Success path — `pg-share-success.compose.yml` (+ `run-success.sh`)
Same setup, plus a runtime edit inside pmm-server: append `listen_addresses = '*'` to
`/srv/postgres14/postgresql.conf` and a `host ... scram-sha-256` rule to
`/srv/postgres14/pg_hba.conf`, then `supervisorctl restart postgresql`. The client (separate
container) then connects over the bridge.

```
./run-success.sh
docker compose -f pg-share-success.compose.yml --profile probe down -v   # cleanup
```
Expected client output: `pg_isready` → `accepting connections` (rc=0); `psql` returns
`pmm-managed` and the bridge server IP, then lists PMM tables.

## Why `EXPOSE`/`ports:` can't substitute for the edit
Neither file publishes 5432 to the host, and neither needs to. The fail case proves that a
localhost-bound service is unreachable from another container regardless of any Compose port
directive — `EXPOSE`/`expose:` is metadata only, and even `ports:` DNATs to the container's
bridge IP (its `eth0`), which a `127.0.0.1`-bound PG still refuses. Reachability is decided by
where the *process* binds, i.e. `listen_addresses`.

## Why `network_mode: "service:pmm"` (shared namespace) is the wrong fix — and a security issue
A tempting shortcut is to put the side container in PMM's network namespace
(`network_mode: "service:pmm"`, i.e. the Compose form of `--network container:pmm-server`).
It "works" on `127.0.0.1:5432` with **zero** PMM changes — but only because it doesn't grant
access to Postgres, it moves the client *inside* PMM's trust boundary. That's a security
problem, not a solution:

- **It exposes everything PMM keeps private, not just Postgres.** PMM binds its internal
  services to `127.0.0.1` precisely so nothing else can reach them (ClickHouse, VictoriaMetrics,
  internal HTTP endpoints, …). Sharing the namespace gives the sidecar loopback-level access to
  *all* of them — including services that trust localhost implicitly and have no auth. The
  localhost bind **is** the security boundary; a shared namespace collapses it wholesale, with no
  way to scope access to only the DB.
- **No isolation or auditability.** The sidecar has no network identity of its own, so this
  access is invisible to any network policy, firewall, or segmentation — it can't be limited to a
  subnet/user or observed as a real endpoint connection.
- **It couples the two containers** (shared port space → collisions on 8080/8443/5432/9090/…;
  the sidecar's networking is tied to PMM's lifecycle) and **doesn't generalize** beyond a single
  Docker host (a separate pod/host/network needs a real endpoint anyway).

The supported, least-privilege alternative is the "success path" above: keep the containers
independently networked, bind PG to a reachable interface (`listen_addresses`), and authorize
**only** the intended client subnet/user (`pg_hba`). Everything else stays private.

## Notes / caveats
- Separate named volumes + project names per file, so the fail case is genuinely unconfigured.
- `0.0.0.0/0` in the pg_hba rule is demo-only; scope to the bridge subnet
  (`docker network inspect pmm-pg-success_pmmnet -f '{{range .IPAM.Config}}{{.Subnet}}{{end}}'`).
- The demo reuses the `pmm-managed` role/DB for brevity. For real use, create a dedicated
  least-privilege role + database instead of handing a side app PMM's owner role.
- Uses `percona/pmm-server:3.8.1`, pinned to `platform: linux/amd64` (no arm64 build yet, so
  it runs emulated on Apple Silicon — expect a slower boot, hence the generous healthcheck).
