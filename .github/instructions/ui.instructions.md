---
applyTo: ui/**
---
# PMM UI Development Guidelines

> **Parent guide**: [PMM_AGENTS.md](../../PMM_AGENTS.md) — product overview, architecture, domain model, global conventions
> **Related**: [api.instructions.md](api.instructions.md) (API definitions consumed by UI) · [managed.instructions.md](managed.instructions.md) (server backend)

The `/ui` directory contains the PMM web frontend — a React/TypeScript application that provides the primary user interface for Percona Monitoring and Management. It runs inside a Grafana iframe on PMM Server and also hosts standalone pages for updates, RTA, and help.

## Architecture

### Monorepo Structure

The UI uses a **Yarn workspaces + Turborepo** monorepo with three packages:

| Package | Path | Purpose |
|---------|------|---------|
| **pmm** | `ui/apps/pmm/` | Main PMM UI application (Vite + React) |
| **pmm-compat** | `ui/apps/pmm-compat/` | Grafana plugin for PMM ↔ Grafana integration (Webpack) |
| **@pmm/shared** | `ui/packages/shared/` | Shared code: cross-frame messaging, types, utilities (Rollup) |

### Key Technology Choices

| Technology | Role |
|------------|------|
| **React 18** | UI framework |
| **TypeScript** | Type-safe development |
| **Vite** | Dev server and production build (main app) |
| **MUI (Material UI)** | Component library |
| **@percona/percona-ui** | Percona's shared UI component library and theme |
| **TanStack Query (React Query)** | Server state management (API caching, mutations) |
| **React Context** | UI/auth state (AuthProvider, SettingsProvider, etc.) |
| **Vitest** | Unit testing (main app) |
| **Jest** | Unit testing (shared package) |
| **Webpack** | Build for Grafana plugin (pmm-compat) |
| **Rollup** | Build for shared package |

### Communication with Grafana

PMM UI runs inside a Grafana iframe. Cross-frame communication uses `CrossFrameMessenger` from `@pmm/shared`:
- Navigation events
- Theme synchronization
- Authentication state

## Directory Structure

```
ui/
├── package.json                     # Yarn workspaces root
├── turbo.json                       # Turborepo pipeline configuration
├── yarn.lock
├── apps/
│   ├── pmm/                         # Main PMM UI application
│   │   ├── src/
│   │   │   ├── main.tsx             # App entry point
│   │   │   ├── App.tsx              # Root component
│   │   │   ├── router.tsx           # Route definitions (createBrowserRouter)
│   │   │   ├── Providers.tsx        # Context provider composition
│   │   │   ├── api/                 # API client functions (axios-based)
│   │   │   ├── components/          # Reusable UI components
│   │   │   ├── contexts/            # React contexts (auth, user, settings, navigation, tour)
│   │   │   ├── hooks/               # Custom hooks (useQuery/useMutation wrappers per API)
│   │   │   ├── icons/               # SVG icon components
│   │   │   ├── lib/                 # Constants, utilities, messenger
│   │   │   ├── pages/               # Route pages
│   │   │   │   ├── updates/         # PMM Server updates
│   │   │   │   ├── update-clients/  # Client updates
│   │   │   │   ├── help-center/     # Help center
│   │   │   │   ├── rta/             # Real-Time Analytics (tabs, sessions, overview)
│   │   │   │   └── not-found/       # 404 page
│   │   │   ├── types/               # TypeScript type definitions
│   │   │   └── utils/               # Utility functions
│   │   ├── vite.config.ts           # Vite build configuration
│   │   └── vitest.config.ts         # Test configuration
│   │
│   └── pmm-compat/                  # Grafana plugin
│       ├── src/
│       │   ├── compat.ts            # Main plugin entry
│       │   ├── lib/                 # Plugin utilities
│       │   ├── theme/               # Theme bridging
│       │   └── styles/              # Plugin styles
│       ├── webpack.config.ts
│       └── .config/                 # Grafana plugin config (provisioning, Dockerfile)
│
└── packages/
    └── shared/                      # Shared library
        ├── src/
        │   ├── messenger.ts         # CrossFrameMessenger for iframe communication
        │   ├── types/               # Shared TypeScript types
        │   └── utils/               # Shared utilities (isRenderingServer, etc.)
        └── rollup.config.js
```

## Routing

Routes are defined in `ui/apps/pmm/src/router.tsx` using React Router's `createBrowserRouter` with `basename: '/pmm-ui'`:

| Route | Page |
|-------|------|
| `/` | Redirects to `/graph` (Grafana) |
| `/updates` | PMM Server updates |
| `/updates/clients` | Client updates |
| `/help` | Help center |
| `/rta` | Real-Time Analytics tab |
| `/rta/selection` | RTA service selection |
| `/rta/sessions` | RTA sessions list |
| `/rta/overview` | RTA overview |
| `/graph/*` | Grafana iframe |
| `*` | 404 fallback |

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

Providers are composed in `Providers.tsx`:
- `AuthProvider` — authentication state
- `UserProvider` — current user info
- `SettingsProvider` — PMM Server settings
- `UpdatesProvider` — update availability
- `GrafanaProvider` — Grafana integration state
- `NavigationProvider` — sidebar navigation
- `TourProvider` — onboarding tour
- `ThemeContextProvider` — theme from `@percona/percona-ui`

## API Layer

API calls are organized in `src/api/` using axios. Each API module provides typed request/response functions that are consumed by custom hooks in `src/hooks/`.

## Patterns and Conventions

### Do
- Use TanStack Query (`useQuery`, `useMutation`) for all server state
- Create custom hooks per API domain in `src/hooks/`
- Use MUI and `@percona/percona-ui` components for consistent styling
- Use TypeScript strict mode — define types in `src/types/`
- Co-locate test files next to components (`*.test.tsx`)
- Use `CrossFrameMessenger` for communication with the Grafana iframe

### Don't
- Don't use Redux or other state management — TanStack Query + Context covers all needs
- Don't bypass React Query for API calls — it handles caching, deduplication, and background refetch
- Don't use CSS-in-JS directly — use MUI's `sx` prop or theme-aware styled components
- Don't hardcode URLs — use constants from `src/lib/constants.ts`
- Don't add Grafana-specific code to the main `pmm` app — use `pmm-compat` for Grafana plugin logic

## Testing

- **Framework**: Vitest (main app), Jest (shared package)
- **Libraries**: `@testing-library/react`, `@testing-library/jest-dom`
- **Setup**: `src/setupTests.ts` provides global mocks (clipboard, `navigator.isSecureContext`)
- **Config**: `vitest.config.ts` — jsdom environment, `globals: true`
- **Pattern**: co-located `*.test.tsx` / `*.test.ts` files next to components
- **Run**: `make test` or via Turborepo (`turbo test`)

## Development Workflow

```bash
# Prerequisites: Node 22, Yarn
cd ui

# Install dependencies
make setup

# Start dev server
make dev

# Production build
make build

# Run tests
make test
```

## Key Files to Reference

- `ui/package.json` — workspace root, scripts, dependencies
- `ui/turbo.json` — Turborepo pipeline configuration
- `ui/apps/pmm/src/router.tsx` — route definitions
- `ui/apps/pmm/src/Providers.tsx` — context provider composition
- `ui/apps/pmm/src/api/` — API client functions
- `ui/apps/pmm/src/hooks/` — React Query hooks per API domain
- `ui/apps/pmm/vite.config.ts` — build configuration
- `ui/packages/shared/src/messenger.ts` — cross-frame communication
