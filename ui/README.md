# Percona Monitoring and Management UI

[Percona Monitoring and Management (PMM)](https://www.percona.com/software/database-tools/percona-monitoring-and-management) is a best-of-breed open source database monitoring solution. It helps you reduce complexity, optimize performance, and improve the security of your business-critical database environments, no matter where they are located or deployed.
PMM helps users to:

- Reduce Complexity
- Optimize Database Performance
- Improve Data Security

See the [PMM Documentation](https://www.percona.com/doc/percona-monitoring-and-management/2.x/index.html) for more information.

## Stack

This repo uses the following stack across its packages:

- Yarn (https://yarnpkg.com/)
- Turborepo (https://turborepo.com/)
- Typescript (https://www.typescriptlang.org/)
- React (https://react.dev/)
- Rollup to bundle the different common packages (https://rollupjs.org/)
- Vite for development (https://vitejs.dev/)
- Vitest for unit tests (https://vitest.dev/)

## Apps

- **pmm** — main PMM UI application
- **pmm-compat** — Grafana plugin that handles communication between Grafana and PMM UI

## Packages

- **shared** — common code between applications

## Run in the devcontainer (recommended)

The PMM devcontainer (see the root `CONTRIBUTING.md`) now ships Node 22 + Yarn and a Vite dev server that runs end-to-end with the rest of PMM Server. From the repo root **on the host**:

```bash
make env-up      # first run only; reuses the container afterwards
make env         # shell into the container
```

Then **inside the container**:

```bash
make run-ui
```

`run-ui` installs UI dependencies, symlinks the `pmm-compat` plugin into Grafana's plugin directory, injects livereload into Grafana's `index.html` (`setup-livereload`), and starts Vite on port `5173`.

Open `https://localhost/` — Grafana loads the `pmm-compat` plugin, which fetches the main UI from the Vite dev server. Edits under `ui/apps/pmm/src/` hot-reload in the browser without a full page refresh.

Notes:

- The Vite port is configurable via `PMM_PORT_VITE` in your `.env` (see `.env.dev.example`); it defaults to `5173`.
- `run-ui` installs an EXIT trap that restores the original `pmm-compat-app` plugin and restarts Grafana when you Ctrl-C. Don't kill the container mid-run, or the restore is skipped.
- For a one-shot build deployed into the container's system paths, use `make run-ui-build` instead.

## Run locally on the host

Use this when you want to drive Vite from your IDE without `make env`. You still need a reachable PMM Server — the simplest way is to leave the devcontainer running (`make env-up`) so its ports are exposed; any other PMM Server reachable at `https://localhost:8443` works too.

Prerequisites:

- [Node 22](https://nodejs.org/en) (e.g. via [nvm](https://github.com/nvm-sh/nvm))
- [Yarn](https://yarnpkg.com/) 1.x

```bash
make setup       # yarn install across the workspace
make dev         # turbo dev → Vite on https://localhost:5174 (or 5173 if nginx certs are present)
```

Vite proxies `/v1` and `/graph` to `https://localhost:8443` (see `apps/pmm/vite.config.ts`). Without nginx certificates available at `/srv/nginx/certificate.{crt,key}`, Vite falls back to port `5174` with a self-signed cert from `@vitejs/plugin-basic-ssl`.

## Build for production

```bash
make build
```

## Other targets

```bash
make test        # vitest across the workspace
make lint
make format
```
