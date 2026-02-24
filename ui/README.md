# Percona Monitoring and Management UI

[Percona Monitoring and Management (PMM)](https://www.percona.com/software/database-tools/percona-monitoring-and-management) is a best-of-breed open source database monitoring solution. It helps you reduce complexity, optimize performance, and improve the security of your business-critical database environments, no matter where they are located or deployed.
PMM helps users to:

- Reduce Complexity
- Optimize Database Performance
- Improve Data Security

See the [PMM Documentation](https://www.percona.com/doc/percona-monitoring-and-management/2.x/index.html) for more information.

## Pre-Requisites

Make sure you have the following installed:

- [node 22](https://nodejs.org/en) (you can also use [nvm](https://github.com/nvm-sh/nvm) to manage node versions)
- [yarn](https://yarnpkg.com/)

## Stack

This repo uses the following stack across its packages:

- Yarn (https://yarnpkg.com/)
- Turborepo (https://turborepo.com/)
- Typescript (https://www.typescriptlang.org/);
- React (https://react.dev/);
- Rollup to bundle the different common packages (https://rollupjs.org/);
- Vite for development (https://vitejs.dev/);
- Vitest for unit tests (https://vitest.dev/);

## Install dependencies

Before installing dependencies, make sure to be logged in to Github Packages, as `@percona/percona-ui` lives as a Github Package due to (hopefully temporary) problems with NPM registry authentication, thus needing for users to be authenticated in order to install it:

- Create a Personal Access Token on Github;
- Add it as a newline to existing `.npmrc`, as so: `//npm.pkg.github.com/:_authToken=${GITHUB_AUTH_TOKEN}`

```bash
make setup
```

## Run in development mode

```bash
make dev
```

## Build application for production

```bash
make build
```

## Apps

- **pmm** - main PMM UI application
- **pmm-compat** - Grafana plugin that handles communication between Grafana and PMM UI

## Packages

- **shared** - common code between applications
