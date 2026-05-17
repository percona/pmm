# PMM — datacosmos build & fork model

This is a datacosmos fork of [`percona/pmm`](https://github.com/percona/pmm),
tracking the upstream **`v3`** branch.

## Branch model

| Branch | Purpose | How to keep current |
|---|---|---|
| `main` | Integration branch — the datacosmos default branch; merges `feat/build` and `feat/clickhouse-collector`. | `git merge upstream/v3` |
| `feat/build` | datacosmos OCI/RPM packaging pipeline (`Makefile.datacosmos`, `datacosmos-release.yml`) + agent coordination protocol. The branch the team builds and releases from. | `git merge upstream/v3` |
| `feat/clickhouse-collector` | Isolated **draft** of a ClickHouse metrics collector. Compilable, not yet upstream-ready. | `git merge upstream/v3` |

Upstream is tracked via the `upstream/v3` remote branch — there is no local
mirror branch. Custom commits live only on the fork branches, so periodic
`git merge upstream/v3` keeps conflicts confined to the fork-specific files.
This mirrors the standard long-lived-fork practice
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
for b in feat/build feat/clickhouse-collector main; do
  git switch "$b" && git merge upstream/v3            # resolve conflicts in fork files only
done
```

## Releases

datacosmos releases are tagged `v3-<ISO date>-<upstream commit count>` — e.g.
`v3-2026-05-17-5580`: the `v3` line, the date the tag was cut, and
`git rev-list --count upstream/v3` (the upstream commit the build is based on).
The old `-dcN` counter is retired.

```bash
make -f Makefile.datacosmos release-tag               # creates the v3-<date>-<count> tag
git push origin v3-<date>-<count>                     # triggers the datacosmos release workflow
```

Pushing the tag runs `.github/workflows/datacosmos-release.yml`, which builds
multi-arch images and a GitHub Release. The build's internal `PMM_VERSION`
(written to `VERSION`, used for the upstream S3 RPM cache) is derived
separately from the nearest upstream semver tag — it stays a clean `X.Y.Z`.

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

### Standalone (no Percona infrastructure)

`Makefile.datacosmos` builds with **no Jenkins, no AWS S3 build cache, no
private/Percona registries, no image push**:

- `RPMBUILD_DOCKER_IMAGE` defaults to the locally-built `pmm-rpmbuild:local`
  (built by `make build-rpmbuild-image` from `oraclelinux:9-slim`).
- `SKIP_S3_CACHE=1` is exported — `build/scripts/build-server-rpm` then builds
  every RPM locally instead of syncing `s3://pmm-build-cache`.
- Images use local tags (`pmm-local/*`) and are never pushed.
- `make prepare` materialises the build tree under `$(ROOT_DIR)` (default
  `../pmm-build-root`, **outside** the repo).

### ⚠️ Build status — verified vs. pending

Verified working: `make build-rpmbuild-image` (local 1.5 GB rpmbuild image),
`make prepare` (submodule checkout + build-tree layout), and the early
`make client` stages (external tarball downloads, source preparation).

**Pending:** the canonical PMM build (orchestrated upstream by
[`Percona-Lab/pmm-submodules`](https://github.com/Percona-Lab/pmm-submodules)
+ Jenkins) assumes `root_dir` is itself a git checkout (the Jenkins workspace)
and writes the Go module cache under `root_dir/tmp`. Reproducing a full
`make client` / `make server` standalone needs `build/scripts/vars` to be
decoupled from that workspace assumption. Until then, run the full RPM/image
build inside a `Percona-Lab/pmm-submodules` checkout, or in CI.

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
