# QAN App (pmm-app) Development Guidelines

> **Parent guide**: [AGENTS.md](../../AGENTS.md) — product overview, architecture, domain model, global conventions
> **Related**: [dashboards/dashboards/AGENTS.md](../dashboards/AGENTS.md) (dashboard JSON definitions bundled by this plugin) · [ui/AGENTS.md](../../ui/AGENTS.md) (main PMM frontend) · [api/AGENTS.md](../../api/AGENTS.md) (API definitions consumed by QAN) · [qan-api2/AGENTS.md](../../qan-api2/AGENTS.md) (QAN backend)

The `dashboards/pmm-app/` directory contains a **Grafana application plugin** (`type: app`, `id: pmm-app`) that bundles PMM dashboard JSON definitions and provides the custom **Query Analytics (QAN) panel** (`pmm-qan-app-panel`). It is built with TypeScript and React on top of Grafana's plugin SDK.

## Architecture

### Plugin Structure

The pmm-app plugin consists of two sub-plugins registered in their respective `plugin.json` manifests:

1. **App plugin** (`pmm-app/src/plugin.json`) — declares the application, registers PMM dashboard JSON includes, and exposes the QAN panel.
2. **Panel plugin** (`pmm-app/src/pmm-qan/plugin.json`) — declares the `pmm-qan-app-panel` panel type used by `Query Analytics/pmm-qan.json`.

```
pmm-app/src/module.ts          → AppPlugin() (minimal app shell)
pmm-app/src/pmm-qan/module.ts  → PanelPlugin(QueryAnalyticsPanel)

plugin.json includes[]:
  - dashboards from dashboards/dashboards/**/*.json
  - panel: pmm-qan-app-panel

Build (webpack) → dist/
  → deployed to Grafana plugins directory on PMM Server
```

### Key Technology Choices

| Technology | Role |
|------------|------|
| **TypeScript** | Type-safe development |
| **React 18** | UI framework |
| **Webpack** | Build tooling (Grafana plugin scaffolding) |
| **Yarn 1.x** | Package manager (`packageManager: yarn@1.22.21`) |
| **SCSS / LESS** | Styling |
| **@grafana/data, @grafana/ui, @grafana/runtime** | Grafana plugin SDK (`>=11.x.x`) |
| **Ant Design** | Additional UI components (QAN panel) |
| **axios** | HTTP client for QAN API calls |
| **react-table** | Table rendering in QAN Overview |
| **d3** | Data visualization |
| **Jest 29** | Unit testing (`@swc/jest`, `jest-environment-jsdom`) |

## QAN Panel

The Query Analytics panel lives in `pmm-app/src/pmm-qan/` and is registered as a `PanelPlugin` wrapping the `QueryAnalytics` React component.

### Key Sub-Components

| Component | Path | Purpose |
|-----------|------|---------|
| **QueryAnalytics** | `pmm-qan/panel/QueryAnalytics.tsx` | Root panel component |
| **Overview** | `pmm-qan/panel/components/Overview/` | Main query table with sortable metrics columns |
| **Details** | `pmm-qan/panel/components/Details/` | Query detail view: Explain, Metrics, Metadata, Table |
| **Filters** | `pmm-qan/panel/components/Filters/` | Filter sidebar (dimension, value filtering) |
| **BarChart** | `pmm-qan/panel/components/BarChart/` | Time-distribution bar chart |
| **ManageColumns** | `pmm-qan/panel/components/ManageColumns/` | Column visibility picker |

### Shared Code

`pmm-app/src/shared/` contains reusable code across the QAN panel:
- `components/` — common UI elements (Table, Modal, Charts, Icons, Form controls)
- `components/helpers/` — humanization, formatting, validators
- `components/hooks/` — shared React hooks (e.g., window size)
- `global-styles/themes/` — dark/light theme SCSS variables

## Patterns and Conventions

### Do
- Co-locate test files next to components (`*.test.tsx`)
- Use `@testing-library/react` for component tests
- Use `@grafana/data` and `@grafana/ui` APIs for Grafana integration
- Use the existing provider pattern in `pmm-qan/panel/provider/` for QAN state
- Follow the Grafana plugin SDK conventions for panel lifecycle

### Don't
- Don't modify files under `.config/` — they are scaffolded by `@grafana/create-plugin` and carry "do not edit" warnings
- Don't introduce new state management libraries — use React state/context as in existing QAN code
- Don't duplicate dashboard JSON inside `pmm-app/src/` — the canonical source is `dashboards/dashboards/`
- Don't bypass the Grafana plugin SDK APIs for data queries or runtime services

## Testing

- **Framework**: Jest 29 with `@swc/jest` transform, `jest-environment-jsdom`
- **Libraries**: `@testing-library/react`, `@testing-library/jest-dom`, `@testing-library/user-event`, `jest-canvas-mock`, `mockdate`
- **Config**: `pmm-app/jest.config.js` extends `pmm-app/.config/jest.config.js`; sets `TZ=GMT`
- **Pattern**: ~35 co-located `*.test.tsx` / `*.test.ts` files under `pmm-app/src/`
- **Run**: `make test` (from `dashboards/`) or `yarn test:ci` (from `pmm-app/`)
- **Linting**: `yarn lint` runs ESLint on `src/**/*.{ts,tsx}`; `yarn typecheck` runs `tsc --noEmit`

## Development Workflow

```bash
# Prerequisites: Node >= 18, Yarn 1.x
cd dashboards/pmm-app

# Install dependencies
yarn install --frozen-lockfile

# Start webpack in watch mode (development)
yarn dev

# Production build
yarn build

# Run tests (CI mode)
yarn test:ci

# Lint and typecheck
yarn lint && yarn typecheck
```

### Docker Development

`pmm-app/docker-compose.yaml` provides a local Grafana environment that mounts `./dist` into the PMM Server plugin directory:

```bash
cd dashboards/pmm-app
docker-compose up -d
yarn dev
```

### Makefile Targets (from `dashboards/`)

| Target | Purpose |
|--------|---------|
| `make install` | `yarn install --frozen-lockfile` in `pmm-app/` |
| `make build` | `yarn build` in `pmm-app/` |
| `make release` | `install` + `build` |
| `make test` | `release` + `yarn test:ci` |
| `make clean` | Remove `pmm-app/dist/` |
| `make install-plugins` | Download ClickHouse and Polystat Grafana plugins |

## Key Files to Reference

- `dashboards/Makefile` — build, test, and release targets
- `dashboards/pmm-app/package.json` — dependencies, scripts, engine requirements
- `dashboards/pmm-app/src/plugin.json` — app plugin manifest (dashboard includes, panel registration)
- `dashboards/pmm-app/src/pmm-qan/plugin.json` — QAN panel plugin manifest
- `dashboards/pmm-app/src/module.ts` — app plugin entry point
- `dashboards/pmm-app/src/pmm-qan/module.ts` — QAN panel entry point
- `dashboards/pmm-app/src/pmm-qan/panel/QueryAnalytics.tsx` — root QAN panel component
- `dashboards/pmm-app/jest.config.js` — test configuration
- `dashboards/pmm-app/docker-compose.yaml` — local development environment
- `dashboards/CONTRIBUTING.md` — contribution workflow and local dev setup
