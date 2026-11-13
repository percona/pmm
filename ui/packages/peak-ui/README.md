<picture>
  <source media="(prefers-color-scheme: dark)" srcset="https://raw.githubusercontent.com/percona/pmm/main/ui/packages/peak-ui/.storybook/static/logo-peak-ui-dark.svg" />
  <img src="https://raw.githubusercontent.com/percona/pmm/main/ui/packages/peak-ui/.storybook/static/logo-peak-ui.svg" alt="Peak UI" height="37" />
</picture>

# Peak UI

Peak UI (`@percona/peak-ui`) is Percona's React component library built on top of [MUI v7](https://mui.com/). It provides themed UI components, design tokens, and [react-hook-form](https://react-hook-form.com/)-integrated form inputs for Percona products, with theme variants, such as PMM (for Percona Monitoring and Management native UI).

Browse the components in [Storybook](https://main--6a31152ea29e3a78e1060595.chromatic.com/).

## Usage

Install the package and its peer dependencies (for date/time picker inputs, also install a date adapter such as `date-fns` and set up MUI X `LocalizationProvider`):

```bash
pnpm add @percona/peak-ui \
  react react-dom \
  @mui/material @mui/icons-material @mui/utils @mui/x-date-pickers \
  @emotion/react @emotion/styled \
  notistack react-hook-form
```

Wrap your app in `ThemeContextProvider` with one of the exported theme options (`baseThemeOptions`, `pmmThemeOptions`, or `sepThemeOptions`), then use the components:

```tsx
import { ThemeContextProvider, pmmThemeOptions, Card } from '@percona/peak-ui';

export const App = () => (
  <ThemeContextProvider themeOptions={pmmThemeOptions}>
    <Card>Hello, Percona</Card>
  </ThemeContextProvider>
);
```

Using an AI coding assistant? The package ships an [`AGENTS.md`](./AGENTS.md) with the usage rules agents need (theme wrapper, form binding, where Peak UI ends and MUI begins). It is readable straight from `node_modules/@percona/peak-ui/AGENTS.md`. Next to it, [`llms.txt`](./llms.txt) indexes every export with a one-line purpose and a link to its documentation.

## Development

Peak UI lives in the [PMM repository](https://github.com/percona/pmm) at `ui/packages/peak-ui`. PMM's apps and plugins use it from source through the pnpm workspace, so a component change and the code that uses it land in one pull request. Contributor conventions live in [`CONTRIBUTING.md`](./CONTRIBUTING.md).

Install once from `ui/`, then run the scripts from this folder, or from `ui/` with `pnpm --filter @percona/peak-ui <script>` (for example `pnpm --filter @percona/peak-ui storybook:dev`).

```bash
cd ui && pnpm install
```

| Task          | Command                          |
| ------------- | -------------------------------- |
| Storybook dev | `pnpm storybook:dev` (port 6006) |
| Build         | `pnpm build` (Rollup → `dist/`)  |
| Build (watch) | `pnpm build:watch`               |
| Typecheck     | `pnpm check-types`               |
| Lint          | `pnpm lint`                      |
| Format check  | `pnpm format:check`              |
| Tests         | `pnpm test`                      |

## Releases

Versions follow PMM: every PMM release tag (`v3.x.y`) publishes `@percona/peak-ui@3.x.y` to npm from GitHub Actions, with provenance. Each version ships a generated `CHANGELOG.md`; a release without Peak UI changes says `No changes to Peak UI`. Nobody publishes by hand.
