# Peak Design

Peak Design (`@percona/percona-ui`) is Percona's React component library built on top of [MUI v7](https://mui.com/). It provides themed UI components, design tokens, and [react-hook-form](https://react-hook-form.com/)-integrated form inputs for Percona products, with theme variants, such as PMM (for Percona Monitoring and Management native UI).

Browse the components in [Storybook](https://percona.github.io/percona-ui/).

## Usage

Install the package and its peer dependencies (for date/time picker inputs, also install a date adapter such as `date-fns` and set up MUI X `LocalizationProvider`):

```bash
pnpm add @percona/percona-ui \
  react react-dom \
  @mui/material @mui/icons-material @mui/utils @mui/x-date-pickers \
  @emotion/react @emotion/styled \
  notistack react-hook-form
```

Wrap your app in `ThemeContextProvider` with one of the exported theme options (`baseThemeOptions`, `pmmThemeOptions`, or `sepThemeOptions`), then use the components:

```tsx
import { ThemeContextProvider, pmmThemeOptions, Card } from '@percona/percona-ui';

export const App = () => (
  <ThemeContextProvider themeOptions={pmmThemeOptions}>
    <Card>Hello, Percona</Card>
  </ThemeContextProvider>
);
```

## Development

```bash
pnpm install
```

| Task          | Command                              |
| ------------- | ------------------------------------ |
| Storybook dev | `pnpm storybook:dev` (port 6006)     |
| Build         | `pnpm build` (Rollup → `dist/`)      |
| Build (watch) | `pnpm build:watch`                   |
| Typecheck     | `pnpm typecheck`                     |
| Lint          | `pnpm lint`                          |
| Format check  | `pnpm format:check`                  |
| Tests         | `pnpm test`                          |

## Publishing

```bash
# 1. Bump the version in package.json (e.g. X.Y.Z)

# 2. Commit the change
git commit -am "release: vX.Y.Z"

# 3. Create an annotated tag
git tag -a vX.Y.Z -m "vX.Y.Z"

# 4. Push the commit and the tag
git push && git push --tags

# 5. Log in to npm (optional if already logged in; 2FA required)
npm login

# 6. Publish
pnpm publish --access public --registry https://registry.npmjs.org/
```
