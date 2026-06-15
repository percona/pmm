# Percona Monitoring and Management UI

[Percona Monitoring and Management (PMM)](https://www.percona.com/software/database-tools/percona-monitoring-and-management) is a best-of-breed open source database monitoring solution. It helps you reduce complexity, optimize performance, and improve the security of your business-critical database environments, no matter where they are located or deployed.
PMM helps users to:

- Reduce Complexity
- Optimize Database Performance
- Improve Data Security

See the [PMM Documentation](https://www.percona.com/doc/percona-monitoring-and-management/2.x/index.html) for more information.

See detailed information about prerequisites and setup [here](../../README.md)

# Locally testing @percona/percona-ui

`@percona/percona-ui` is a normal npm dependency (see `package.json`). To iterate on the library and PMM together, `yarn link` an in-progress checkout against this app. The recipe depends on whether you're running `make dev` on the host or `make run-ui` inside the PMM devcontainer.

In both cases:

- Check out the lib from https://github.com/percona/percona-ui.
- After linking, **uncomment** the `exclude` block in `vite.config.ts` so Vite stops pre-bundling the linked package:
  ```ts
  // exclude: ['@percona/percona-ui'],
  ```
- When you're done, **comment the `exclude` block back**, then from `ui/apps/pmm`:
  ```bash
  yarn unlink @percona/percona-ui
  yarn install --force
  ```
- Restarting the dev server between linking/unlinking is advised.

## Host-local flow (`make dev`)

- From the lib folder on the host: `pnpm build:watch` and `yarn link`.
- From `ui/apps/pmm` on the host: `yarn link @percona/percona-ui`.
- Any change in the lib triggers a rebuild and HMR in PMM.

## Devcontainer flow (`make run-ui`)

The host's `yarn link` global registry isn't visible inside the devcontainer, so the link has to happen there. Two options:

**Bind-mount a host checkout** — keeps the lib editable from your host IDE:

1. Clone `percona-ui` alongside `pmm` on the host (so it sits at `../percona-ui` relative to the repo root).
2. Uncomment the volume mapping in `docker-compose.dev.yml`
3. `make env-up` (or `make env-up-rebuild` if the container was already running) then `make env` from the host.
4. Inside the container:
   ```bash
   cd /root/go/src/github.com/percona/percona-ui
   yarn install
   yarn link
   pnpm build:watch &       # leave the watcher running
   cd /root/go/src/github.com/percona/pmm/ui/apps/pmm
   yarn link @percona/percona-ui
   ```
5. Uncomment the `exclude` block in `vite.config.ts`, then back at the repo root: `make run-ui`.
