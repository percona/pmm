# Percona Monitoring and Management UI

[Percona Monitoring and Management (PMM)](https://www.percona.com/software/database-tools/percona-monitoring-and-management) is a best-of-breed open source database monitoring solution. It helps you reduce complexity, optimize performance, and improve the security of your business-critical database environments, no matter where they are located or deployed.
PMM helps users to:

- Reduce Complexity
- Optimize Database Performance
- Improve Data Security

See the [PMM Documentation](https://www.percona.com/doc/percona-monitoring-and-management/2.x/index.html) for more information.

See detailed information about prerequisites and setup [here](../../README.md)

# Locally testing @percona/percona-ui

`@percona/percona-ui` is a normal npm dependency (see `package.json`). To iterate on the library and PMM together, `pnpm link` an in-progress checkout against this app. The recipe depends on whether you're running `make dev` on the host or `make run-ui` inside the PMM devcontainer.

In both cases:

- Check out the lib from https://github.com/percona/percona-ui.
- After linking, **uncomment** the `exclude` block in `vite.config.ts` so Vite stops pre-bundling the linked package:
  ```ts
  // exclude: ['@percona/percona-ui'],
  ```
- When you're done, **comment the `exclude` block back**, then from `ui/apps/pmm`:
  ```bash
  pnpm unlink @percona/percona-ui
  pnpm install
  ```
- Restarting the dev server between linking/unlinking is advised.

`pnpm link` takes a path, so the same command works on the host and inside the devcontainer — there is no global link registry to share between them.

## Host-local flow (`make dev`)

1. Clone `percona-ui` alongside `pmm` on the host (so it sits at `../percona-ui` relative to the repo root).
2. From the lib folder: `pnpm install && pnpm build:watch` — leave the watcher running.
3. From `ui/apps/pmm`: `pnpm link ../../../../percona-ui`.
4. Any change in the lib triggers a rebuild and HMR in PMM.

## Devcontainer flow (`make run-ui`)

**Bind-mount a host checkout** — keeps the lib editable from your host IDE:

1. Clone `percona-ui` alongside `pmm` on the host (so it sits at `../percona-ui` relative to the repo root).
2. Uncomment the volume mapping in `docker-compose.dev.yml`
3. Run `make env-up` then `make env` from the host.
4. Inside the container:
   ```bash
   cd /root/go/src/github.com/percona/percona-ui
   pnpm install
   pnpm build:watch &       # leave the watcher running
   cd /root/go/src/github.com/percona/pmm/ui/apps/pmm
   pnpm link /root/go/src/github.com/percona/percona-ui
   ```
5. Uncomment the `exclude` block in `vite.config.ts`, then back at the repo root: `make run-ui`.
