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
- For a one-shot build deployed into the container's system paths, use `make build-ui` instead.

### Update Grafana in the devcontainer

The devcontainer ships a prebuilt Grafana baked into the `perconalab/pmm-server` dev image. To develop against a local [percona/grafana](https://github.com/percona/grafana) fork instead, mount your checkout into the container and rebuild it:

1. Clone the Grafana fork **next to** the `pmm` repo on the host, so it resolves to `../grafana` from the repo root:

   ```bash
   git clone https://github.com/percona/grafana ../grafana
   ```

2. Uncomment the `grafana` volume mappings in `docker-compose.dev.yml`:

   ```yaml
   # grafana
   - ../grafana:/root/go/src/github.com/percona/grafana
   - ../grafana/public:/usr/share/grafana/public
   ```

   The first mount provides the Grafana source for the backend build; the second serves the fork's built frontend (`public/`).

3. Recreate the container so the new mounts take effect — volume mappings are read at container create time (`make env-down` then `make env-up`, or recreate via your container tooling).

4. Rebuild the Grafana backend **inside the container**:

   ```bash
   make grafana-be-build
   ```

   This runs `make build-go` in `/root/go/src/github.com/percona/grafana`, copies the resulting `bin/linux/amd64/grafana` binary to `/usr/sbin/grafana`, and restarts Grafana via supervisor.

For frontend changes in the fork, rebuild its `public/` assets (`make build-js` inside the grafana checkout); they are served through the `../grafana/public` mount.

## Run locally on the host

Use this when you want to drive Vite from your IDE without `make env`. You still need a reachable PMM Server — the simplest way is to leave the devcontainer running (`make env-up`) so its ports are exposed; any other PMM Server reachable at `https://localhost:8443` works too.

Prerequisites:

- [Node 22](https://nodejs.org/en) (e.g. via [nvm](https://github.com/nvm-sh/nvm))
- [Yarn](https://yarnpkg.com/) 1.x

```bash
make setup       # yarn install across the workspace
make dev         # turbo dev → Vite on https://localhost:5174 (or 5173 if nginx certs are present)
```

Vite proxies `/v1`, `/graph`, and `/logs.zip` to the PMM Server (inside the devcontainer: `https://localhost:8443`; on the host when using the devcontainer-exposed ports: `https://localhost`) — see `apps/pmm/vite.config.ts`.

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
