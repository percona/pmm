# AGENTS.md

## Cursor Cloud specific instructions

This is a zero-infrastructure React component library (`@percona/percona-ui`). No databases, Docker, or external services are needed.

### Quick reference

| Task | Command |
|------|---------|
| Install deps | `pnpm install` |
| Build | `pnpm build` |
| Typecheck | `pnpm typecheck` |
| Lint | `pnpm lint` |
| Format check | `pnpm format:check` |
| Unit tests | `pnpm test` |
| Storybook dev | `pnpm storybook:dev` (port 6006) |
| Storybook build | `pnpm storybook:build` |

### Non-obvious caveats

- pnpm 10 warns about "ignored build scripts" for esbuild. The esbuild binary still resolves correctly from the pnpm store; this warning is cosmetic and does not block Vite/Storybook/Vitest.
- `CLAUDE.md` states "no linter or test runner configured" but that is outdated — `pnpm lint` (ESLint) and `pnpm test` (Vitest) both work. Refer to `package.json` scripts as the source of truth.
- Rollup build emits a TS5096 warning (`allowImportingTsExtensions`) and a circular-dependency note — both are pre-existing and non-blocking.
- Storybook uses `--no-open` flag already; it starts headlessly on port 6006.
