# AGENTS.md

Guidance for any coding agent (Humans, Claude Code, Cursor, Codex, etc.). This is the single source of truth. `CLAUDE.md` imports it.

Percona UI (`@percona/percona-ui`) is a React + MUI v7 component library published as an npm package (themed components, design tokens, form inputs). Zero infrastructure: no DB/Docker/services needed to build, test, or run Storybook.

## Commands

- **Build:** `pnpm build` (Rollup → `dist/`)
- **Build watch:** `pnpm build:watch`
- **Storybook dev:** `pnpm storybook:dev` (port 6006, headless via `--no-open`)
- **Storybook build:** `pnpm storybook:build`
- **Type check:** `pnpm typecheck` (app + `.storybook` tsconfigs)
- **Lint:** `pnpm lint` (ESLint over `src`)
- **Format:** `pnpm format` (Prettier write) · **check only:** `pnpm format:check`
- **Test:** `pnpm test` (Vitest, run once) · **watch:** `pnpm test:watch`

## Verification (MANDATORY after any code change)

Before considering the work done, run the safety gate and make it pass:

```
pnpm typecheck && pnpm lint && pnpm format:check && pnpm test
```

If `format:check` fails, fix it with `pnpm format` (don't hand-edit whitespace). If `typecheck`, `lint`, or `test` fail, fix the underlying issue and re-run the full chain until green. Do not report a change as complete until this passes.

### Known non-blocking warnings

Pre-existing and cosmetic — do not treat as failures:
- `pnpm build`: a TS5096 warning (`allowImportingTsExtensions`) and a circular-dependency note.
- `pnpm install`: pnpm 10 reports "ignored build scripts" for esbuild; the binary still resolves correctly and doesn't block Vite/Storybook/Vitest.

## Architecture (non-obvious bits)

- **Themes** — three variants (`base`, `pmm`, `sep`) under `src/design/themes/`. `getThemeOptions(themeName)` is **curried** — call it as `getThemeOptions(name)(mode)` (the return value is `(mode) => ThemeOptions`). PMM/SEP extend Base via `mergeThemeOptions` (`src/design/merge-theme-options.ts`).
- **ThemeContextProvider** wraps MUI's `ThemeProvider` with a light/dark toggle (`ColorModeContext`), persisting mode to localStorage.
- **Components** — follow the existing folder layout for new ones: component file + `.types.ts` + `.stories.tsx` + `index.ts` barrel, with **both** named and default exports. Form inputs (`src/components/form/inputs/`) integrate with `react-hook-form`.

## Tech Stack (beyond what package.json shows)

React 18 + TypeScript strict (`react-jsx` transform). Rollup bundles to ESM with sourcemaps; peer deps (MUI, Emotion, React, `react-hook-form`) are externalized. Fonts via `@fontsource` (Poppins, Roboto, Roboto Mono).

## Path Aliases

- `@/utils` → `./src/utils` (configured in tsconfig.json)

## MUI Conventions

### Imports — prefer path imports over barrels

Barrel imports (`@mui/material`, `@mui/icons-material`) hurt dev startup/rebuild most, not just bundle size. Use default path imports (matches MUI docs).

- **Components:** `import Button from '@mui/material/Button'` — not `import { Button } from '@mui/material'`.
- **Icons:** `import Delete from '@mui/icons-material/Delete'` — not the barrel.
- **Migrate existing barrels:** `npx @mui/codemod@latest v5.0.0/path-imports <path>`.
- **Optional ESLint guard:** `no-restricted-imports` with `{ "regex": "^@mui/[^/]+$" }` blocks package-root barrels only.
- **VS Code nudge:** `typescript.preferences.autoImportSpecifierExcludeRegexes: ["^@mui/[^/]+$"]` in `.vscode/settings.json`.

### Theme overrides — gotchas

- **Child themes use `mergeThemeOptions`, not raw `deepmerge`.** PMM/SEP extend BaseTheme via `mergeThemeOptions` (`src/design/merge-theme-options.ts`): it composes `styleOverrides` (child wins per-key, base slots survive) and concatenates `variants` base-first (child cascades last). Use `mergeThemeOptions(baseThemeOptions(mode), newOptions)`; never raw `deepmerge`. Array slots other than `variants` are still replaced — extend `mergeThemeOptions` if a new array slot must compose.
- **MUI v7 drives hover backgrounds via CSS custom properties.** E.g. `IconButton`'s hover bg is `var(--IconButton-hoverBg)`, set per-variant. Override by setting the CSS var on the right slot (e.g. `"--IconButton-hoverBg": theme.palette.action.hover` in `colorSecondary`), not `"&:hover": { backgroundColor: ... }`. Grep `--{Component}-` in MUI source first.

## Design Constraints (non-obvious — verify before "normalizing")

### Chip

- Outlined uses `palette.[color].light` for text + border — except `warning`, which uses `warning.main`. `warning.light` is a yellow/gold hue with insufficient contrast on light backgrounds; `main` is the smallest step that holds. Don't normalize warning to `light` — it's a perceptual constraint.
- Filled uses the `palette.[color].surface` token (not MUI-standard). Don't refactor `surface` to `.light`/`.main` — those are reserved for borders/icons and have a different perceptual role.

### NavItem

The `<Box sx={{ mr: -1.75 }} />` rendered when no `icon` is passed is intentional — it preserves text start-line alignment with icon-bearing rows.

### Tooltip

- Tooltips render in the **inverted** app mode (light app → dark tooltip surface, dark app → light tooltip surface). Don't add per-theme tooltip overrides — change `BaseTheme` instead where possible.
- Link styling inside tooltips lives as a nested `& .MuiLink-root` rule *inside* the tooltip override, **not** in `MuiLink.styleOverrides`. Don't move link styles back to `MuiLink`.

### Action token model (state colors)

Two parallel layers, intentionally distinct — don't collapse them:

- **`palette.action.{active,hover,selected,disabled,focus}`** = neutral state tints, brand-black-tinted in all themes; used wherever the surface is default-colored.
- **`palette.primary.{hover,selected,focus,focusVisible,outlinedBorder}`** = state tints for *primary-colored* surfaces. Identical to `action.*` in Base (both brand-black) but diverges in PMM/SEP (purple).
- Per-mode opacity is mandatory: dark mode uses higher alpha (8% hover vs 4% light, 16% selected vs 8% light, 15% disabled/focus vs 12% light), wired through `tokens.action.{hoverOpacity,selectedOpacity,disabledOpacity,focusOpacity}` so MUI's `alpha(...)` calculations stay in sync.

### Theme extension contract

PMM and SEP extend BaseTheme with `mergeThemeOptions` (see MUI Conventions). Child themes may override any `styleOverrides` slot and add `variants` — both compose with Base, so baseline styles are always respected and the child wins only on genuine conflicts. Shared behavior belongs in BaseTheme; brand-specific tweaks belong in PMM/SEP.

## Code Comments (CRITICAL)

- Prefer no comment; code should be self-explanatory. One short line where a comment is genuinely needed — never paragraph-style docblocks for obvious logic.
- If a comment needs more than ~150 characters, refactor or move the explanation to the PR/commit — not source.
- Comment only non-obvious caveats worth flagging. Applies to `//` and `/* */` in TypeScript/TSX.
