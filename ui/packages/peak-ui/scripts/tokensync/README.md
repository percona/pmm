## What it is

Utility to synchronize design tokens from a Figma export (in W3C Design Tokens format) into our TypeScript theme files for Percona UI. By running the script, we automate the process of resolving variable references and updating our codebase with the latest design choices from Figma.

## How to use it

1. Export all Figma variables, using the plugin "Luckino - Variables Import/Export JSON & CSS", into 1 file called `variables.json` formatted to the standard W3C Design Tokens, and place it in this folder
2. Run the `sync.mjs` script:
  ```bash
    node scripts/tokensync/sync.mjs             # apply changes
    node scripts/tokensync/sync.mjs --dry-run   # preview only
  ```

## What the script does

1. Reads `variables.json` and resolves all Figma variable references (including cross-references like `{basis.neutral.main}` and `{basis.action.focus}`)
2. Generates TypeScript code preserving primitive references (e.g., `primitives.brand.stone[800]`) instead of flattening to raw hex values
3. Replaces these `export const` blocks in `BaseTheme.tsx`: `primitives`, `semanticTokensLight`, `semanticTokensDark`, `defaultPrimaryLight`, `defaultPrimaryDark`
4. Replaces these `export const` blocks in `PmmTheme.tsx`: `pmmPrimaryLight`, `pmmPrimaryDark`

### Implemented mapping


| **Figma JSON**                                 | **Mode**     | **Theme file target**                                    |
| ---------------------------------------------- | ------------ | -------------------------------------------------------- |
| `primitives.`                                  | —            | `primitives` (BaseTheme)                                 |
| `basis.{surfaces,text,lines,action,error,...}` | Light        | `semanticTokensLight` (BaseTheme)                        |
| `basis.{surfaces,text,lines,action,error,...}` | Dark         | `semanticTokensDark` (BaseTheme)                         |
| `basis.primary.neutral`                        | Light / Dark | `defaultPrimaryLight` / `defaultPrimaryDark` (BaseTheme) |
| `basis.primary.pmm`                            | Light / Dark | `pmmPrimaryLight` / `pmmPrimaryDark` (PmmTheme)          |


### To be mapped


| **Variable**                    | **Value(s)**                                                        |
| ------------------------------- | ------------------------------------------------------------------- |
| `basis.surfaces.elevationX`     | Light: `rgba(44, 50, 62, 0.04)` / Dark: `rgba(240, 241, 244, 0.08)` |
| `tokens.shape.borderRadiusXs`   | `3`                                                                 |
| `tokens.shape.borderRadiusSm`   | `5`                                                                 |
| `tokens.shape.borderRadiusMd`   | `8`                                                                 |
| `tokens.shape.borderRadiusLg`   | `12`                                                                |
| `tokens.shape.borderRadiusXl`   | `18`                                                                |
| `tokens.shape.borderRadiusFull` | `999`                                                               |
| `tokens.shape.borderRadius`     | `5` (alias for `borderRadiusSm`)                                    |


