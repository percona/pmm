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
 * Browser-facing mount point of the SEP backend inside PMM.
 *
 * pmm-server's nginx exposes the SEP side-car under this one location, so
 * every SEP surface the SPA calls (`/api`, `/stream-logs`, `/execution-events`,
 * `/files`, …) is reached as `${SEP_BASE_PATH}/<route>`. SEP owns the generic
 * top-level names it would otherwise claim — `/api` in particular, which PMM is
 * the natural claimant for.
 *
 * The prefix is *not* stripped in front of SEP: SEP serves it itself via
 * `root_path`. Stripping it was measured and rejected — with `root_path` unset,
 * `request.url_for()` emits prefix-less absolute URLs inside JSON payloads, so
 * the links SEP hands back would escape the prefix. A dev backend therefore has
 * to run with `root_path` set the same way the shipped side-car does.
 */
export const SEP_BASE_PATH = '/sep';
