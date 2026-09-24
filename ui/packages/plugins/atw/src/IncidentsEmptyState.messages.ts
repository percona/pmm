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
 * Generic PMM docs root until a dedicated Support diagnostics page ships.
 * Same short-link used before the setup gate was removed (PMM-15405).
 */
export const SUPPORT_DIAGNOSTICS_DOCS_URL =
  'https://per.co.na/pmm_documentation';

export const Messages = {
  title: 'No incidents yet',
  description:
    'An incident is a workspace for one support case. Create one, run diagnostic snippets against your monitored databases, and review outputs in Results — nothing is sent until you choose Send.',
  create: 'New incident',
  // Label matches the target: the generic docs root, not a diagnostics page.
  documentation: 'PMM documentation',
};
