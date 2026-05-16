# PMM — datacosmos build & fork model

This is a datacosmos fork of [`percona/pmm`](https://github.com/percona/pmm),
aligned to the upstream stable tag **`v3.7.1`**.

## Branch model

| Branch | Base | Purpose | How to keep current |
|---|---|---|---|
| `v3` | upstream `v3.7.1` | Pristine mirror of upstream — **never edit directly**. | `git fetch upstream && git merge upstream/v3` |
| `datacosmos/build` | `v3.7.1` | datacosmos OCI/RPM packaging pipeline (`Makefile.datacosmos`) + agent coordination protocol. The branch the team builds from. | `git merge v3` |
| `feat/clickhouse-collector` | `v3.7.1` | Isolated **draft** of a ClickHouse metrics collector. Compilable, not yet upstream-ready. | rebase onto `v3` before any upstream PR |

Custom commits are never placed on `v3`, so future upstream syncs stay
conflict-free. This mirrors the standard long-lived-fork practice
([Atlassian](https://www.atlassian.com/git/tutorials/git-forks-and-upstreams),
[GitHub Docs](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/working-with-forks/syncing-a-fork)).

## Remotes

```
origin    https://github.com/datacosmos-br/pmm.git   # the fork
upstream  https://github.com/percona/pmm.git          # Percona
```

## Syncing with upstream (periodic)

```bash
git fetch upstream --tags
git switch v3 && git merge --ff-only upstream/v3      # or merge a newer tag
git switch datacosmos/build && git merge v3           # resolve conflicts in custom files only
```

## Building (datacosmos pipeline)

`Makefile.datacosmos` is **included** by the root `Makefile` and drives an
RPM/OCI build via `build/scripts/build-client` and `build/scripts/build-server`.

```bash
make all          # prepare + rpmbuild image + client + server
make client       # client RPM/DEB/Docker only
make server       # server RPM/Docker only
make publish      # collect artifacts into ./artifacts
make clean
```

### ⚠️ Build caveats (validate in CI — not verifiable offline)

- The **canonical** PMM build is orchestrated by the separate
  [`Percona-Lab/pmm-submodules`](https://github.com/Percona-Lab/pmm-submodules)
  repo (`ci.yml` + Jenkins). `Makefile.datacosmos` is a *local alternative*
  that calls `build/scripts/*` directly — it must be re-validated whenever the
  fork is realigned to a new upstream tag, since those scripts evolve.
- `build-rpmbuild-image`, `client`, `server` run Docker/`buildx` and push to
  `gru.ocir.io/grq1iurfepyg/pmm`. They cannot be dry-run in a plain checkout.
- `prepare` depends on the `pmm-submodules` git submodule being initialised
  (`git submodule update --init --recursive`).

## ClickHouse collector (draft — `feat/clickhouse-collector`)

`agent/agents/clickhouse/{collector.go,config.go}` is a `prometheus.Collector`
draft. It **compiles** (after `go get github.com/ClickHouse/clickhouse-go/v2`)
but is **not wired into pmm-agent** and is **not upstream-ready**. Before any
PR to `percona/pmm` it needs: integration into pmm-agent's exporter framework,
English comments, configuration via `pmm-agent.yaml`, and tests.
It is intentionally kept off `datacosmos/build` to avoid carrying the
`clickhouse-go/v2` dependency into the build before the feature is real.

## Agent coordination

This repo adopts the mandatory multi-agent coordination protocol —
see `.agents/skills/agent-coordination/SKILL.md` and the live ledger
`.agents/coordination/LEDGER.md`. Details in `AGENTS.md` §W-9.
