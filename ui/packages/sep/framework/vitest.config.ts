/**
 * Copyright (C) 2026 Percona LLC
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <https://www.gnu.org/licenses/>.
 */

import react from '@vitejs/plugin-react';
import { defineConfig } from 'vitest/config';

export default defineConfig({
  plugins: [react()],
  resolve: {
    // Collapse to one instance of each. Under pnpm, react-dom and the code under
    // test otherwise resolve `react` to different symlink paths, giving vite-node
    // two React modules → "Invalid hook call". `dedupe` funnels every specifier to
    // one resolved copy (no hardcoded node_modules paths, which break under pnpm).
    dedupe: ['react', 'react-dom', 'react-hook-form', '@tanstack/react-query'],
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./tests/setup.ts'],
    css: false,
    server: {
      deps: {
        // Route React consumers through Vite's transform so their `react` imports
        // share the single instance.
        inline: [
          'react-dom',
          '@testing-library/react',
          '@percona/percona-ui',
          '@mui/material',
          '@mui/system',
          '@mui/utils',
          'material-react-table',
        ],
      },
    },
  },
});
