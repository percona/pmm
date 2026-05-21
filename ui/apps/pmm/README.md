# Percona Monitoring and Management UI

[Percona Monitoring and Management (PMM)](https://www.percona.com/software/database-tools/percona-monitoring-and-management) is a best-of-breed open source database monitoring solution. It helps you reduce complexity, optimize performance, and improve the security of your business-critical database environments, no matter where they are located or deployed.
PMM helps users to:

- Reduce Complexity
- Optimize Database Performance
- Improve Data Security

See the [PMM Documentation](https://www.percona.com/doc/percona-monitoring-and-management/2.x/index.html) for more information.

See detailed information about prerequisites and setup [here](../../README.md)

# Locally testing @percona/percona-ui

- Checkout code from https://github.com/percona/percona-ui
- From the lib folder, run `pnpm build:watch` and `yarn link`
- On this repo's `ui/apps/pmm` folder, run `yarn link @percona/percona-ui` and uncomment the `exclude` block from `vite.config.ts`
- Any change on the lib will trigger a build and a refresh to PMM
- When you're done testing, comment back the `exclude` block and then, from `ui/apps/pmm` again: `yarn unlink @percona/percona-ui` and `yarn install --force`
- Restarting dev server between linking/unlinking is advised
