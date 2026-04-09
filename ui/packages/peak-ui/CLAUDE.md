# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

Percona UI (`@percona/percona-ui`) — a React component library built on MUI v7, published as an npm package. It provides themed UI components, design tokens, and form inputs for Percona products. Licensed AGPL-3.0-or-later.

## Commands

- **Build:** `pnpm build` (Rollup → `dist/`)
- **Build watch:** `pnpm build:watch`
- **Storybook dev:** `pnpm storybook:dev` (port 6006)
- **Storybook build:** `pnpm storybook:build`
- **Design token sync:** `node scripts/tokensync/sync.mjs` (or `--dry-run` to preview)

There is no linter or test runner configured in package.json scripts.

## Architecture

### Entry Point

`src/index.ts` re-exports three modules: `components`, `design`, and `utils`.

### Design System (`src/design/`)

- **ThemeContextProvider** — wraps MUI's `ThemeProvider` with light/dark mode toggle via React context (`ColorModeContext`). Persists color mode to localStorage.
- **Themes** — two theme variants:
  - `base` — default Percona theme (`src/design/themes/base/`)
  - `pmm` — PMM (Percona Monitoring and Management) theme (`src/design/themes/pmm/`)
  - `sep` — SEP (Services Enablement Platform) theme (`src/design/themes/sep/`)
- **`getThemeOptions(themeName)(mode)`** — factory that returns MUI `ThemeOptions` for a given theme name and palette mode.
- **Design tokens** — primitives and semantic tokens (light/dark) are defined in theme files, synced from Figma via `scripts/tokensync/sync.mjs`.

### Components (`src/components/`)

All components are React + TypeScript, built on MUI. Key patterns:
- Each component folder typically has: component file, `.types.ts` for prop types, `.stories.tsx` for Storybook, and an `index.ts` barrel export.
- Form inputs (`src/components/form/inputs/`) integrate with `react-hook-form`.
- Table component wraps `material-react-table`.
- Components use both named and default exports from their barrel files.

### Foundations (`src/foundations/`)

Storybook-only documentation pages for colors and typography — not exported in the library.

### Utils (`src/utils/`)

Currently exports string utilities only.

## Tech Stack

- **React 18** with TypeScript (strict mode, `react-jsx` transform)
- **MUI v7** (`@mui/material`, `@mui/icons-material`, `@mui/x-date-pickers`)
- **Emotion** for styling
- **Rollup** for library bundling (ESM output with sourcemaps, peer deps externalized)
- **Storybook 8** for component development and documentation
- **react-hook-form** for form state management
- **material-react-table** for table component
- **Fonts:** Poppins, Roboto, Roboto Mono (via `@fontsource`)

## Path Aliases

- `@/utils` → `./src/utils` (configured in tsconfig.json)

## Design Token Sync

The `scripts/tokensync/` workflow syncs Figma design tokens into theme TypeScript files:
1. Export Figma variables as W3C Design Tokens JSON (`variables.json`)
2. Run `node scripts/tokensync/sync.mjs` to update `BaseTheme.tsx` and `PmmTheme.tsx`
3. The script preserves primitive references (e.g., `primitives.brand.stone[800]`) rather than flattening to hex values

## Storybook

Storybook supports theme switching (base/pmm) and color mode switching (light/dark) via toolbar globals. Stories are organized: Introduction → Foundations (Colors, Typography, Icons) → Components.
