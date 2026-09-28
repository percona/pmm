# Grafana Dashboards Development Guidelines

> **Parent guide**: [AGENTS.md](../AGENTS.md) — product overview, architecture, domain model, global conventions
> **Related**: [ui/apps/pmm-app/AGENTS.md](../ui/apps/pmm-app/AGENTS.md) (Grafana plugin that bundles these dashboards) · [managed/AGENTS.md](../managed/AGENTS.md) (server backend providing metrics data)

The `dashboards/` directory contains Grafana dashboard JSON definitions organized by database and domain area. These are the canonical source for all PMM monitoring dashboards. Besides the JSON folders it only holds the `misc/` Python helper scripts (importing, exporting, converting, and cleaning up dashboard JSON) and the packaging metadata the RPM spec references (`LICENSE`, `README.md`).

## Architecture

Dashboard JSON files are standard Grafana dashboard exports. They are loaded into Grafana through two mechanisms:

1. **Plugin bundling** — `ui/apps/pmm-app/src/plugin.json` declares each dashboard in its `includes` array. `ui/apps/pmm-app/src/dashboards` is a symlink to this folder; the pmm-app build copies the JSON files into the plugin `dist/dashboards/` directory.
2. **Grafana provisioning** — the PMM Server Ansible role configures Grafana to load dashboards from the plugin's `dist/dashboards/` path.

```
dashboards/**/*.json
  → ui/apps/pmm-app/src/dashboards (symlink)
    → pmm-app build (copied to dist/dashboards/)
      → Grafana provisioning on PMM Server
        → Grafana UI (visualization)
```

In the devcontainer, `make run-ui` runs pmm-app's `scripts/watch-dashboards.mjs`, which mirrors every dashboard JSON change into the provisioned dashboards path. The dev compose file shortens Grafana's provisioning poll to 5s (`dev/grafana-dashboards-provisioning.yml`), so edits show up without a restart or a manual reload.

## Dashboard categories

| Directory | Domain |
|-----------|--------|
| `MySQL/` | MySQL, PXC/Galera, Aurora, ProxySQL, HAProxy |
| `MongoDB/` | MongoDB, WiredTiger, MMAPv1, InMemory, PBM, ReplSet |
| `PostgreSQL/` | PostgreSQL, Patroni |
| `OS/` | Node, CPU, memory, disk, network, NUMA, processes |
| `Valkey/` | Valkey/Redis clients, cluster, memory, replication, slowlog |
| `Insight/` | Home Dashboard, Advanced Data Exploration, VictoriaMetrics, Exporters |
| `Experimental/` | Databases Overview, DB Cluster Summary |
| `PMM Health/` | Environments Overview, PMM Health Overview, PMM HA Health Overview |
| `Query Analytics/` | QAN panel wrapper (`pmm-qan.json`) |
| `Kubernetes (experimental)/` | Kubernetes operator monitoring |

## Patterns and Conventions

### Do
- Design dashboards in the Grafana UI, then export the JSON
- Use `misc/cleanup-dash.py` to normalize exported JSON before committing
- Run `python3 -m unittest discover -s dashboards/misc -p 'test_*.py'` from the repo root after changing a dashboard or the cleanup script; it re-checks every dashboard in the tree
- Follow the existing directory structure when adding dashboards for a new domain
- Keep one dashboard per JSON file, named to match the dashboard title
- Register new dashboards in `ui/apps/pmm-app/src/plugin.json` under the `includes` array — CI (`ui.yml`) fails if an `includes` path doesn't resolve

### Don't
- Don't edit dashboard JSON by hand unless making targeted fixes — use the Grafana UI for design work
- Don't duplicate dashboard JSON inside `ui/apps/pmm-app/src/` — the canonical source is this folder
- Don't commit Grafana-generated volatile fields (e.g., `version`, `iteration`) — use `cleanup-dash.py` to strip them
- Don't format dashboard JSON with oxfmt/Prettier — `cleanup-dash.py` owns its layout. oxfmt is told to skip it through the root `.prettierignore` (`/dashboards/**/*.json`) and `ui/.oxfmtrc.json` (the same files reached through the pmm-app `src/dashboards` symlink); keep both entries if either path changes
- Don't add anything else at the root of this folder beyond `misc/`, `LICENSE`, and `README.md` — it holds dashboard JSON only

## Key Files to Reference

- `dashboards/` — all dashboard JSON definitions, organized by domain
- `dashboards/misc/cleanup-dash.py` — JSON normalizer enforced by CI (`dashboards.yml`)
- `ui/apps/pmm-app/src/plugin.json` — plugin manifest that registers dashboards in Grafana
- `dashboards/README.md` — featured dashboards list
- `ui/apps/pmm-app/CONTRIBUTING.md` — contribution workflow and local dev setup
