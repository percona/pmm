# QAN App (pmm-app) Development Guidelines

> **Parent guide**: [AGENTS.md](../../../AGENTS.md) — product overview, architecture, domain model, global conventions
> **Related**: [dashboards/AGENTS.md](../../../dashboards/AGENTS.md) (dashboard JSON definitions bundled by this plugin) · [ui/AGENTS.md](../../AGENTS.md) (main PMM frontend, monorepo this app is part of) · [api/AGENTS.md](../../../api/AGENTS.md) (API definitions consumed by QAN) · [qan-api2/AGENTS.md](../../../qan-api2/AGENTS.md) (QAN backend)

The `ui/apps/pmm-app/` directory contains a **Grafana application plugin** (`type: app`, `id: pmm-app`) that bundles PMM dashboard JSON definitions and provides the custom **Query Analytics (QAN) panel** (`pmm-qan-app-panel`). It is built with TypeScript and React on top of Grafana's plugin SDK, and is a workspace member of the `ui/` Yarn workspaces + Turborepo monorepo (see [ui/AGENTS.md](../../AGENTS.md)).

## Architecture

### Plugin Structure

The pmm-app plugin consists of two sub-plugins registered in their respective `plugin.json` manifests:

1. **App plugin** (`src/plugin.json`) — declares the application, registers PMM dashboard JSON includes, and exposes the QAN panel.
2. **Panel plugin** (`src/pmm-qan/plugin.json`) — declares the `pmm-qan-app-panel` panel type used by `Query Analytics/pmm-qan.json`.

```
src/module.ts          → AppPlugin() (minimal app shell)
src/pmm-qan/module.ts  → PanelPlugin(QueryAnalyticsPanel)
src/dashboards          → symlink to the top-level dashboards/ folder

plugin.json includes[]:
  - dashboards from dashboards/**/*.json (resolved through the src/dashboards symlink)
  - panel: pmm-qan-app-panel

Build (webpack) → dist/
  → deployed to Grafana plugins directory on PMM Server
```

### Key Technology Choices

| Technology                                       | Role                                                                       |
| ------------------------------------------------ | -------------------------------------------------------------------------- |
| **TypeScript**                                   | Type-safe development                                                      |
| **React 18**                                     | UI framework                                                               |
| **Webpack**                                      | Build tooling (Grafana plugin scaffolding)                                 |
| **Yarn 1.x**                                     | Package manager, hoisted from the `ui/` workspace root                     |
| **SCSS / LESS**                                  | Styling                                                                    |
| **@grafana/data, @grafana/ui, @grafana/runtime** | Grafana plugin SDK (`>=11.x.x`)                                            |
| **Ant Design**                                   | Additional UI components (QAN panel)                                       |
| **axios**                                        | HTTP client for QAN API calls                                              |
| **react-table**                                  | Table rendering in QAN Overview                                            |
| **d3**                                           | Data visualization                                                         |
| **Jest 29**                                      | Unit testing (`@swc/jest`, `jest-environment-jsdom`)                       |
| **chokidar**                                     | Dashboard JSON file watcher for local dev (`scripts/watch-dashboards.mjs`) |

## QAN Panel

The Query Analytics panel lives in `src/pmm-qan/` and is registered as a `PanelPlugin` wrapping the `QueryAnalytics` React component.

### Key Sub-Components

| Component          | Path                                      | Purpose                                              |
| ------------------ | ----------------------------------------- | ---------------------------------------------------- |
| **QueryAnalytics** | `pmm-qan/panel/QueryAnalytics.tsx`        | Root panel component                                 |
| **Overview**       | `pmm-qan/panel/components/Overview/`      | Main query table with sortable metrics columns       |
| **Details**        | `pmm-qan/panel/components/Details/`       | Query detail view: Explain, Metrics, Metadata, Table |
| **Filters**        | `pmm-qan/panel/components/Filters/`       | Filter sidebar (dimension, value filtering)          |
| **BarChart**       | `pmm-qan/panel/components/BarChart/`      | Time-distribution bar chart                          |
| **ManageColumns**  | `pmm-qan/panel/components/ManageColumns/` | Column visibility picker                             |

### Shared Code

`src/shared/` contains reusable code across the QAN panel:

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
- Don't duplicate dashboard JSON inside `src/` — the canonical source is the top-level `dashboards/` folder, reached through the `src/dashboards` symlink
- Don't bypass the Grafana plugin SDK APIs for data queries or runtime services

## Testing

- **Framework**: Jest 29 with `@swc/jest` transform, `jest-environment-jsdom`
- **Libraries**: `@testing-library/react`, `@testing-library/jest-dom`, `@testing-library/user-event`, `jest-canvas-mock`, `mockdate`
- **Config**: `jest.config.js` extends `.config/jest.config.js`; sets `TZ=GMT`
- **Pattern**: ~35 co-located `*.test.tsx` / `*.test.ts` files under `src/`
- **Run**: `yarn test` (one-shot) or `yarn test:watch` (watch mode) from `ui/apps/pmm-app/`, or `turbo run test --filter=pmm-app` from `ui/`
- **Linting**: `yarn lint` runs ESLint on `src/**/*.{ts,tsx}` (non-mutating; use `yarn lint:fix` to auto-fix); `yarn typecheck` runs `tsc --noEmit`

## Development Workflow

```bash
# Prerequisites: Node >= 18, Yarn 1.x
cd ui/apps/pmm-app

# Install dependencies (or from ui/ to install the whole workspace)
yarn install

# Start webpack in watch mode + the dashboard JSON watcher (development)
yarn dev

# Production build
yarn build

# Run tests (one-shot)
yarn test

# Lint and typecheck
yarn lint && yarn typecheck
```

`yarn dev` runs two processes concurrently: webpack in watch mode with `webpack-livereload-plugin` (QAN JS/TS changes), and `scripts/watch-dashboards.mjs`, a chokidar watcher that mirrors dashboard JSON changes (reached through the `src/dashboards` symlink) into the Grafana-provisioned dashboards directory — both update automatically with no extra commands.

### Docker Development

`docker-compose.yaml` provides a local Grafana environment that mounts `./dist` into the PMM Server plugin directory:

```bash
cd ui/apps/pmm-app
docker-compose up -d
yarn dev
```

### From the `ui/` monorepo

Since this app is a `ui/` workspace member, the standard monorepo commands also work: `cd ui && yarn dev` (or `make run-ui` inside the devcontainer) starts this app's `dev` script alongside `pmm` and `pmm-compat` via Turborepo; `yarn build`/`yarn lint`/`yarn test` similarly fan out via `turbo run <task> --filter=pmm-app` when scoped.

## Key Files to Reference

- `ui/apps/pmm-app/package.json` — dependencies, scripts, engine requirements
- `ui/apps/pmm-app/src/plugin.json` — app plugin manifest (dashboard includes, panel registration)
- `ui/apps/pmm-app/src/pmm-qan/plugin.json` — QAN panel plugin manifest
- `ui/apps/pmm-app/src/module.ts` — app plugin entry point
- `ui/apps/pmm-app/src/pmm-qan/module.ts` — QAN panel entry point
- `ui/apps/pmm-app/src/pmm-qan/panel/QueryAnalytics.tsx` — root QAN panel component
- `ui/apps/pmm-app/jest.config.js` — test configuration
- `ui/apps/pmm-app/docker-compose.yaml` — local development environment
- `ui/apps/pmm-app/scripts/watch-dashboards.mjs` — dashboard JSON live-sync watcher
- `ui/apps/pmm-app/CONTRIBUTING.md` — contribution workflow and local dev setup
