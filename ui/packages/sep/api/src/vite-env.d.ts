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

// Minimal ImportMeta declaration so @sep/api can read `import.meta.env.DEV`
// without depending on Vite directly. Consuming apps (e.g. @sep/shell) pull
// in the full `vite/client` types via their own vite-env.d.ts.
declare global {
  interface ImportMetaEnv {
    readonly DEV: boolean;
    readonly PROD: boolean;
    readonly MODE: string;
    // Opt-in mock-fallback flag — set at build time (e.g. via
    // `VITE_MOCK_API=true vite build`) to enable mock data fallbacks in
    // production-mode bundles such as the Playwright preview target.
    readonly VITE_MOCK_API?: string;
  }

  interface ImportMeta {
    readonly env: ImportMetaEnv;
  }
}

export {};
