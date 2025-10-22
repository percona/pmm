/**
 * Why this file exists:
 * - Our tests import UI code that depends on browser-only APIs (MUI, postMessage, etc.).
 * - Vitest (Node) needs a browser-like environment (jsdom) and proper module resolution for MUI and local path aliases.
 * - MUI sometimes performs "directory imports" (e.g. '@mui/material/CssBaseline') which Node ESM cannot resolve.
 *
 * What this config fixes:
 * 1) Enables a DOM-like environment for React/MUI via `environment: 'jsdom'`.
 * 2) Makes Vitest treat common testing globals (`describe`, `it`, `vi`, `expect`) without imports.
 * 3) Forces all dependencies to be inlined/transformed by Vite so that MUI "directory imports"
 *    get rewritten properly even when they appear inside other packages (e.g. @percona/design).
 * 4) Adds minimal path aliases used by the app in tests (utils, lib, hooks, types, etc.).
 * 5) Adds a safety alias for CssBaseline to avoid Node ESM "directory import" resolution errors.
 *
 * Notes:
 * - If Vitest later warns that `deps.inline` is deprecated, switch to `server.deps.inline` (example below).
 */
import { defineConfig } from 'vitest/config';
import { resolve } from 'path';

export default defineConfig({
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['src/setupTests.ts'],
    deps: {
      inline: true,
    },
  },
  resolve: {
    conditions: ['browser', 'module', 'import', 'default'],
    alias: {
      utils: resolve(__dirname, 'src/utils'),
      lib: resolve(__dirname, 'src/lib'),
      hooks: resolve(__dirname, 'src/hooks'),
      themes: resolve(__dirname, 'src/themes'),
      components: resolve(__dirname, 'src/components'),
      contexts: resolve(__dirname, 'src/contexts'),
      pages: resolve(__dirname, 'src/pages'),
      api: resolve(__dirname, 'src/api'),
      icons: resolve(__dirname, 'src/icons'),
      types: resolve(__dirname, 'src/types'),
      assets: resolve(__dirname, 'src/assets'),
      '@mui/material/CssBaseline': '@mui/material/node/CssBaseline/index.js',
    },
  },
});
