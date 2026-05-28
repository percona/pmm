# Grafana Dashboards Development Guidelines

> **Parent guide**: [AGENTS.md](../../AGENTS.md) — product overview, architecture, domain model, global conventions
> **Related**: [dashboards/pmm-app/AGENTS.md](../pmm-app/AGENTS.md) (Grafana plugin that bundles these dashboards) · [managed/AGENTS.md](../../managed/AGENTS.md) (server backend providing metrics data)

The `dashboards/dashboards/` directory contains Grafana dashboard JSON definitions organized by database and domain area. These are the canonical source for all PMM monitoring dashboards. The `dashboards/misc/` directory provides Python helper scripts for importing, exporting, and converting dashboard JSON files.

## Architecture

Dashboard JSON files are standard Grafana dashboard exports. They are loaded into Grafana through two mechanisms:

1. **Plugin bundling** — `pmm-app/src/plugin.json` declares each dashboard in its `includes` array. The build copies them into the plugin `dist/` directory, and Grafana provisions them when the pmm-app plugin is loaded.
2. **Grafana provisioning** — the PMM Server Ansible role configures Grafana to load dashboards from the plugin's `dist/dashboards/` path.

```
dashboards/dashboards/*.json
  → pmm-app build (copied to dist/dashboards/)
    → Grafana provisioning on PMM Server
      → Grafana UI (visualization)
```

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
| `PMM Health/` | Environments Overview, PMM Health Overview, HA Health Overview |
| `Query Analytics/` | QAN panel wrapper (`pmm-qan.json`) |
| `Kubernetes (experimental)/` | Kubernetes operator monitoring |

## Patterns and Conventions

### Do
- Design dashboards in the Grafana UI, then export the JSON
- Use `misc/cleanup-dash.py` to normalize exported JSON before committing
- Follow the existing directory structure when adding dashboards for a new domain
- Keep one dashboard per JSON file, named to match the dashboard title
- Register new dashboards in `pmm-app/src/plugin.json` under the `includes` array

### Don't
- Don't edit dashboard JSON by hand unless making targeted fixes — use the Grafana UI for design work
- Don't duplicate dashboard JSON inside `pmm-app/src/` — the canonical source is `dashboards/dashboards/`
- Don't commit Grafana-generated volatile fields (e.g., `version`, `iteration`) — use `cleanup-dash.py` to strip them

## Key Files to Reference

- `dashboards/dashboards/` — all dashboard JSON definitions, organized by domain
- `dashboards/pmm-app/src/plugin.json` — plugin manifest that registers dashboards in Grafana
- `dashboards/README.md` — featured dashboards list
- `dashboards/CONTRIBUTING.md` — contribution workflow and local dev setup
