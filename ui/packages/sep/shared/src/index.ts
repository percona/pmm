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

/**
 * @sep/shared — Tiny constants & utility package.
 *
 * API client, auth, types, and schema components have moved to:
 *   - @sep/api      — client, auth, types, hooks
 *   - @sep/framework — SchemaFormRenderer, SchemaDrivenPlugin, etc.
 *
 * This package now only holds cross-cutting constants and tiny utilities
 * shared across all packages (e.g., route paths, feature flags).
 */

// ── Route paths (single source of truth for the sidebar) ──────────────
//
// navigation.tsx (the sidebar) reads every `to:` from this map. router.tsx
// still declares its own `path` literals (it is NOT wired to this map), so
// each value here SHOULD match the corresponding router path and, for
// schema-driven plugins, that plugin's `PLUGIN_BASE_PATH`. Nothing enforces
// that at compile time — the `sidebar-navigation` e2e spec is the safety net
// that DETECTS drift (it cannot prevent it). When migrating a plugin, update
// the router route and the entry here together (see SEP-1270). Convention:
//   • schema-driven plugins live under /apps/<name>  (checksums, mysql_backups, archives)
//   • domain-grouped backups keep /backups/<db>          (mongodb, postgresql)
//   • cross-cutting tools stay at bare top-level paths   (inventory, tasks, …)
// `schemaAlters` backs the "Schema Change / Alters" sidebar item; the other
// /schema-change/* entries (`schemaChecksums`, `schemaInventory`) are legacy
// router aliases with no sidebar consumer, kept intentionally.
export const ROUTES = {
  dashboard: '/',
  login: '/login',
  inventory: '/inventory',
  tasks: '/tasks',
  snippets: '/snippets',
  atw: '/atw',
  alertTemplates: '/alerts/templates',
  alertTroubleshooting: '/alerts/troubleshooting',
  schemaAlters: '/schema-change/alters',
  schemaChecksums: '/schema-change/checksums',
  schemaInventory: '/schema-change/inventory',
  checksums: '/apps/checksums',
  mysqlBackups: '/apps/mysql_backups',
  backupsMongodb: '/backups/mongodb',
  backupsPostgresql: '/backups/postgresql',
  archive: '/apps/archives',
  dipper: '/dipper',
  reports: '/reports',
  settings: '/settings',
} as const;

// ── App-wide constants ────────────────────────────────────────────────
export const APP_NAME = 'Services Enablement Platform';
export const APP_SHORT_NAME = 'SEP';
