# PMM UI Development Guidelines

> **Parent guide**: [AGENTS.md](../AGENTS.md) — product overview, architecture, domain model, global conventions
> **Related**: [api/AGENTS.md](../api/AGENTS.md) (API definitions consumed by UI) · [managed/AGENTS.md](../managed/AGENTS.md) (server backend)

The `/ui` directory contains the PMM web frontend — a React/TypeScript application that provides the primary user interface for Percona Monitoring and Management. It is the outer shell served at `/pmm-ui` on PMM Server: it embeds Grafana in an iframe and hosts its own pages for updates, RTA, and help.

## Architecture

### Monorepo Structure

The UI uses a **pnpm workspaces + Turborepo** monorepo:

| Package                         | Path                                | Purpose                                                                         |
| ------------------------------- | ----------------------------------- | ------------------------------------------------------------------------------- |
| **pmm**                         | `ui/apps/pmm/`                      | Main PMM UI application (Vite + React)                                          |
| **pmm-compat**                  | `ui/apps/pmm-compat/`               | Grafana plugin for PMM ↔ Grafana integration (Webpack)                          |
| **@pmm/shared**                 | `ui/packages/shared/`               | Shared code: cross-frame messaging, types, utilities (Rollup)                   |
| **@pmm/plugins-om**             | `ui/packages/plugins/om/`           | MongoDB Operations app (`OmApp`), mounted by `pmm` under its own routes         |
| **@pmm-extensions/plugins-atw** | `ui/packages/plugins/atw/`          | Support diagnostics app (`AtwApp`), mounted by `pmm` under its own routes       |
| **@pmm-extensions/framework**   | `ui/packages/extensions/framework/` | Building blocks for PMM Extensions pages: schema-driven forms, lists, selectors |
| **@pmm-extensions/api**         | `ui/packages/extensions/api/`       | HTTP client, React Query setup and OpenAPI types for the Extensions backend     |
| **@pmm-extensions/shared**      | `ui/packages/extensions/shared/`    | Route constants and small utilities shared by the Extensions packages           |

The plugin and extension packages are part of PMM and follow this guide like `pmm` does. Their own code may predate a pattern here, so before copying a neighbouring page in a plugin, check how `apps/pmm` solves the same problem.

### Key Technology Choices

| Technology                       | Role                                                 |
| -------------------------------- | ---------------------------------------------------- |
| **React 18**                     | UI framework                                         |
| **TypeScript**                   | Type-safe development                                |
| **Vite**                         | Dev server and production build (main app)           |
| **MUI (Material UI)**            | Component library                                    |
| **@percona/peak-ui**             | Percona's shared UI component library and theme      |
| **TanStack Query (React Query)** | Server state management (API caching, mutations)     |
| **React Context**                | UI/auth state (AuthProvider, SettingsProvider, etc.) |
| **Vitest**                       | Unit testing (main app)                              |
| **Jest**                         | Unit testing (shared package)                        |
| **Webpack**                      | Build for Grafana plugin (pmm-compat)                |
| **Rollup**                       | Build for shared package                             |
| **pnpm (via Corepack)**          | Package manager and workspaces                       |
| **Turborepo**                    | Task runner across the workspace                     |
| **oxlint**                       | Linting (`ui/oxlintrc.json`)                         |
| **oxfmt**                        | Formatting (`ui/.oxfmtrc.json`)                      |

### Communication with Grafana

PMM UI is the top frame and Grafana runs inside its `#grafana-iframe`; the `pmm-compat` plugin runs on the Grafana side and targets `window.top`. Cross-frame communication uses `CrossFrameMessenger` from `@pmm/shared`:

- Navigation events
- Theme synchronization
- Authentication state

## Routing

Routes are defined in `ui/apps/pmm/src/router.tsx` using React Router's `createBrowserRouter` with `basename: '/pmm-ui'`:

| Route                                     | Page                                        |
| ----------------------------------------- | ------------------------------------------- |
| `/`                                       | Redirects to `/graph` (Grafana)             |
| `/updates`                                | PMM Server updates                          |
| `/updates/clients`                        | Client updates                              |
| `/help`                                   | Help center                                 |
| `/alerting/status`                        | Fired alerts                                |
| `/settings/:tab?`                         | PMM Server settings                         |
| `/inventory/services`, `/inventory/nodes` | Inventory                                   |
| `/inventory/{services,nodes}/:id/agents`  | Agents of a service or node                 |
| `/inventory/services/:id/edit`            | Edit a service's labels and RDS credentials |
| `/rta`                                    | Real-Time Analytics tab                     |
| `/rta/selection`                          | RTA service selection                       |
| `/rta/sessions`                           | RTA sessions list                           |
| `/rta/overview`                           | RTA overview                                |
| `/advisors`                               | Advisors                                    |
| `/advisors/insights`                      | Advisor insights                            |
| `/advisors/runs`                          | Advisor runs                                |
| `/advisors/scheduler`                     | Advisor schedules                           |
| `/graph/*`                                | Grafana iframe                              |
| `*`                                       | Page not found (`NotFoundPage`)             |

Pages migrated from Grafana keep their old `/graph/...` URLs working: a fallback route in `router.tsx` redirects them to the native page, and `isMigratedPage` (`src/contexts/grafana/grafana.utils.ts`) keeps the iframe hidden when Grafana navigates there.

## State Management

### Server State (TanStack Query)

API data is managed with React Query hooks:

```typescript
// Custom hooks wrap useQuery/useMutation per API endpoint
const { data: services } = useServices(params);
const { data: user } = useUser();
const { data: settings } = useSettings();
```

Query keys follow the pattern: `['domain:action', params]` (e.g., `['services:list', params]`, `['services:getTypes']`).

### UI/Auth State (React Context)

Providers are composed in `Providers.tsx`, all wrapped by `ThemeContextProvider` (theme from `@percona/peak-ui`) in `App.tsx`:

- `AuthProvider` — authentication state
- `UserProvider` — current user info
- `SettingsProvider` — PMM Server settings
- `UpdatesProvider` — update availability
- `VersionProvider` — reloads the page after the server is upgraded externally
- `GrafanaProvider` — Grafana integration state
- `NavigationProvider` — sidebar navigation
- `TourProvider` — onboarding tour

## API Layer

API calls are organized in `src/api/` using axios. Each API module provides typed request/response functions that are consumed by custom hooks in `src/hooks/`.

## UI Component Library

`@percona/peak-ui` ("Peak Design") is PMM's shared component library — MUI v7-based themed components, design tokens, and `react-hook-form`-integrated inputs, exposing a `pmmThemeOptions` theme variant. **Browse the catalog before hand-rolling a component:**

- **Storybook (component catalog):** https://percona.github.io/peak-ui
- **Source & theme options:** https://github.com/percona/peak-ui
- **Agent guide, matching the installed version:** `node_modules/@percona/peak-ui/AGENTS.md` (usage rules) and `llms.txt` (every export, one line each), in any package that depends on peak-ui, such as `ui/apps/pmm/`

The app is wrapped in `ThemeContextProvider` (see `App.tsx`); style with the theme-aware `sx` prop rather than ad-hoc CSS.

`@percona/percona-ui` is peak-ui's former name. Import from `@percona/peak-ui`; lint rejects the old name.

## Tables

Every data table follows the pattern of Inventory's Nodes page (`apps/pmm/src/pages/inventory/nodes/Nodes.tsx`):

- **Component:** peak-ui `Table`, which wraps Material React Table (MRT) with PMM's theme, empty states (`noDataMessage`, `emptyFilterResultsMessage`) and column visibility persisted under `tableName`. Don't build a table with MRT's `useMaterialReactTable` or `MaterialReactTable`; lint rejects them. MRT types and toolbar components are fine to import.
- **URL state:** filters, search, sorting and pagination live in the URL through `useTableUrlState` (`apps/pmm/src/hooks/utils/useTableUrlState.ts`); outside `apps/pmm`, call peak-ui's `usePerconaTableUrlState` with React Router's `useSearchParams`. Give each table on a page its own `paramPrefix`, spread `tableProps` into `Table`, and pass other controlled state, such as `rowSelection`, through `additionalState`. Don't hand-roll URL parameters for table state.
- **Filters:** column filters on the table, not rows filtered before they reach `data`: `filterVariant: 'select'` with `filterFn: 'equals'`, or `'multi-select'` for several values. Search is the table's global filter; set `enableGlobalFilter: false` on columns it shouldn't match.
- **Toolbar toggles** (such as Nodes' "PMM Server only"): a column filter on a hidden column, toggled through `tableProps.onColumnFiltersChange`.
- **Links into a filtered table:** build the parameter names with `buildTableUrlParamKeys`, as `getFilteredTabUrl` in `apps/pmm/src/pages/inventory/Inventory.utils.ts` does.

Two traps:

- `multi-select` defaults to `arrIncludesSome`, which matches substrings on a string column (`prod` matches `preprod`). Give such columns an exact-match `filterFn`.
- The URL state treats an empty string as no filter, so a "none" or "unassigned" option needs a non-empty value.

## Patterns and Conventions

### Do

- Use TanStack Query (`useQuery`, `useMutation`) for all server state
- Create custom hooks per API domain in `src/hooks/`
- Use MUI and `@percona/peak-ui` components for consistent styling — browse the [Storybook catalog](https://percona.github.io/peak-ui) before building a component from scratch
- Use TypeScript strict mode — define types in `src/types/`
- Co-locate test files next to components (`*.test.tsx`)
- Use `CrossFrameMessenger` for communication with the Grafana iframe
- Build native pages on the shared `Page` wrapper (`src/components/page`); it renders on the paper surface by default, so don't pass `surface="paper"`. Opt out with `surface="canvas"` only for a page composed of cards (Help center, Updates), where the darker tone makes the cards stand out

### Don't

- Don't use Redux or other state management — TanStack Query + Context covers all needs
- Don't bypass React Query for API calls — it handles caching, deduplication, and background refetch
- Don't use CSS-in-JS directly — use MUI's `sx` prop or theme-aware styled components
- Don't hardcode URLs — use constants from `src/lib/constants.ts`
- Don't add Grafana-specific code to the main `pmm` app — use `pmm-compat` for Grafana plugin logic

## Testing

- **Framework**: Vitest (`pmm`, plugin and extension packages), Jest (`pmm-compat`, `@pmm/shared`)
- **Libraries**: `@testing-library/react`, `@testing-library/jest-dom`
- **Setup**: `src/setupTests.ts` in `pmm` provides global mocks (clipboard, `navigator.isSecureContext`); the plugin and extension packages use `tests/setup.ts`, and `pmm-compat` uses `jest-setup.js`
- **Config**: each package's `vitest.config.ts` (jsdom environment, `globals: true`) or `jest.config.js`
- **Pattern**: co-located `*.test.tsx` / `*.test.ts` files next to components
- **Run**: `make test` or via Turborepo (`turbo test`)

## Linting and Formatting

- **Linter**: oxlint, configured in `ui/oxlintrc.json`. `plugins` lists every
  built-in plugin the rules rely on — setting it replaces oxlint's default set,
  and a rule whose plugin is missing is silently inert.
- **Restricted imports**: `no-restricted-imports` enforces the peak-ui rules
  above. Files that predate it are exempted in the config's `overrides`; drop an
  entry once its file moves to peak-ui, and don't add new ones.
- **Formatter**: oxfmt, configured in `ui/.oxfmtrc.json`. It owns formatting;
  there is no Prettier and no formatting rule in the linter.
- **Scope**: build/test config files (`vite.config.ts`, `vitest.config.ts`,
  `webpack.config.ts`, `jest.config.js`) are linted like any other source. The
  only exclusion is `apps/pmm-compat/.config/`, Grafana's auto-generated plugin
  scaffold, which upstream regenerates and tells you not to edit.
- **Run**: `make lint`, `make format` (or `make format-check`, which is what CI
  runs in `.github/workflows/ui.yml`).

## Development Workflow

```bash
# Prerequisites: Node 22. pnpm comes from Corepack, which `make setup`
# enables — the version is pinned by `packageManager` in ui/package.json,
# so never install pnpm separately.
cd ui

# Enable Corepack + install dependencies
make setup

# Start dev server
make dev

# Production build
make build

# Run tests
make test

# Lint (oxlint) and format (oxfmt)
make lint
make format        # make format-check in CI
```

Inside the PMM devcontainer (`make env-up` then `make env` from the repo root), `make run-ui` (main UI HMR via Vite on port 5173) and `make run-qan-ui` (QAN livereload on port 35730) replace `make dev` and wire the dev servers into the bundled Grafana automatically. See `ui/README.md` for details.

## Key Files to Reference

- `ui/package.json` — workspace root, scripts, dependencies
- `ui/turbo.json` — Turborepo pipeline configuration
- `ui/apps/pmm/src/router.tsx` — route definitions
- `ui/apps/pmm/src/Providers.tsx` — context provider composition
- `ui/apps/pmm/src/api/` — API client functions
- `ui/apps/pmm/src/hooks/` — React Query hooks per API domain
- `ui/apps/pmm/vite.config.ts` — build configuration
- `ui/packages/shared/src/messenger.ts` — cross-frame communication
