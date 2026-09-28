# QAN App (pmm-app) Development Guidelines

> **Parent guide**: [AGENTS.md](../../../AGENTS.md) — product overview, architecture, domain model, global conventions
> **Related**: [ui/AGENTS.md](../../AGENTS.md) (the `ui/` monorepo this app belongs to) · [dashboards/AGENTS.md](../../../dashboards/AGENTS.md) (dashboard JSON definitions bundled by this plugin) · [api/AGENTS.md](../../../api/AGENTS.md) (API definitions consumed by QAN) · [qan-api2/AGENTS.md](../../../qan-api2/AGENTS.md) (QAN backend)

The `ui/apps/pmm-app/` directory contains a **Grafana application plugin** (`type: app`, `id: pmm-app`) that bundles PMM dashboard JSON definitions and provides the custom **Query Analytics (QAN) panel** (`pmm-qan-app-panel`). It is built with TypeScript and React on top of Grafana's plugin SDK, and is a member of the `ui/` pnpm workspace + Turborepo monorepo — tooling (pnpm, oxlint, oxfmt) is shared with the other apps, see [ui/AGENTS.md](../../AGENTS.md).

## Architecture

### Plugin Structure

The pmm-app plugin consists of two sub-plugins registered in their respective `plugin.json` manifests:

1. **App plugin** (`src/plugin.json`) — declares the application, registers PMM dashboard JSON includes, and exposes the QAN panel.
2. **Panel plugin** (`src/pmm-qan/plugin.json`) — declares the `pmm-qan-app-panel` panel type used by `Query Analytics/pmm-qan.json`.

```
src/module.ts          → AppPlugin() (minimal app shell)
src/pmm-qan/module.ts  → PanelPlugin(QueryAnalyticsPanel)
src/dashboards         → symlink to the top-level dashboards/ folder

plugin.json includes[]:
  - dashboards/**/*.json (resolved through the src/dashboards symlink)
  - panel: pmm-qan-app-panel

Build (webpack) → dist/
  → deployed to Grafana plugins directory on PMM Server
```

### Key Technology Choices

| Technology                                       | Role                                                                  |
| ------------------------------------------------ | --------------------------------------------------------------------- |
| **TypeScript**                                   | Type-safe development                                                 |
| **React 18**                                     | UI framework                                                          |
| **Webpack**                                      | Build tooling (Grafana plugin scaffolding)                            |
| **SCSS / LESS**                                  | Styling                                                               |
| **@grafana/data, @grafana/ui, @grafana/runtime** | Grafana plugin SDK (`>=11.x.x`)                                       |
| **Ant Design**                                   | Additional UI components (QAN panel)                                  |
| **axios**                                        | HTTP client for QAN API calls                                         |
| **react-table**                                  | Table rendering in QAN Overview                                       |
| **d3**                                           | Data visualization                                                    |
| **Jest 29**                                      | Unit testing (`@swc/jest`, `jest-environment-jsdom`)                  |
| **chokidar**                                     | Dashboard JSON watcher for local dev (`scripts/watch-dashboards.mjs`) |

### React types

The app stays on React 18 while `apps/pmm` uses React 19. Dependencies that don't declare an `@types/react` peer (antd, `rc-*`, `react-final-form`, `@tippyjs/react`) would otherwise pick up the workspace-hoisted `@types/react` 19, so `tsconfig.json` pins `react`/`react-dom` types to this package's own 18.x types through `paths`. Keep that mapping when touching the tsconfig.

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

- Don't modify files under `.config/` — they are scaffolded by `@grafana/create-plugin` and carry "do not edit" warnings (the one local change is dropping the ESLint webpack plugin, since the workspace lints with oxlint)
- Don't introduce new state management libraries — use React state/context as in existing QAN code
- Don't duplicate dashboard JSON inside `src/` — the canonical source is the top-level `dashboards/` folder, reached through the `src/dashboards` symlink
- Don't bypass the Grafana plugin SDK APIs for data queries or runtime services

## Testing

- **Framework**: Jest 29 with `@swc/jest` transform, `jest-environment-jsdom`
- **Libraries**: `@testing-library/react`, `@testing-library/jest-dom`, `@testing-library/user-event`, `jest-canvas-mock`, `mockdate`
- **Config**: `jest.config.js` extends `.config/jest.config.js`; sets `TZ=GMT`
- **Pattern**: ~35 co-located `*.test.tsx` / `*.test.ts` files under `src/`
- **Run**: `pnpm test` (one-shot) or `pnpm test:watch` from `ui/apps/pmm-app/`; `make test` from `ui/` runs it with the rest of the workspace
- **Typecheck**: `pnpm typecheck` runs `tsc --noEmit`; the production build also type-checks through `fork-ts-checker-webpack-plugin`

## Development Workflow

```bash
# From ui/ — installs the whole workspace, including this app
cd ui
make setup

cd apps/pmm-app

# Webpack watch (livereload on port 35730) + dashboard JSON watcher
pnpm dev

# Production build
pnpm build

# Tests, lint (oxlint, non-mutating) and typecheck
pnpm test
pnpm lint
pnpm typecheck
```

`pnpm dev` runs two processes: webpack in watch mode with `webpack-livereload-plugin` (QAN JS/TS changes), and `scripts/watch-dashboards.mjs`, which mirrors dashboard JSON changes into the directory Grafana provisions dashboards from (`/usr/share/percona-dashboards/panels/pmm-app/dist/dashboards`, override with `PERCONA_DASHBOARDS_DIST_DIR`). When that directory isn't writable — typically on the host, outside the devcontainer — the watcher logs a hint and exits, and webpack keeps running.

Inside the devcontainer, `make run-ui` (from the repo root) is the usual entry point: it symlinks this app's `dist/` into Grafana's plugin directory and starts every workspace app's `dev` script. See [CONTRIBUTING.md](CONTRIBUTING.md).

### Docker Development

`docker-compose.yaml` provides a local Grafana environment that mounts `./dist` into the PMM Server plugin directory:

```bash
cd ui/apps/pmm-app
docker-compose up -d
pnpm dev
```

## Key Files to Reference

- `ui/apps/pmm-app/package.json` — dependencies, scripts, engine requirements
- `ui/apps/pmm-app/src/plugin.json` — app plugin manifest (dashboard includes, panel registration)
- `ui/apps/pmm-app/src/pmm-qan/plugin.json` — QAN panel plugin manifest
- `ui/apps/pmm-app/src/module.ts` — app plugin entry point
- `ui/apps/pmm-app/src/pmm-qan/module.ts` — QAN panel entry point
- `ui/apps/pmm-app/src/pmm-qan/panel/QueryAnalytics.tsx` — root QAN panel component
- `ui/apps/pmm-app/jest.config.js` — test configuration
- `ui/apps/pmm-app/scripts/watch-dashboards.mjs` — dashboard JSON live-sync watcher
- `ui/apps/pmm-app/docker-compose.yaml` — local development environment
- `ui/apps/pmm-app/CONTRIBUTING.md` — contribution workflow and local dev setup
- `build/packages/rpm/server/SPECS/percona-dashboards.spec` — RPM that packages this app's `dist/`
