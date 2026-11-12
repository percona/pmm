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

import type { NO_GROUPING } from '../components/GroupBySelect';

export type ClusterGrouping = 'env_name' | 'health' | typeof NO_GROUPING;

/** A grouping that names a column, so not "none". */
export type GroupingColumn = Exclude<ClusterGrouping, typeof NO_GROUPING>;

/** The columns the toolbar filters drive. */
export type FilterColumn = 'health' | 'type' | 'env_name';
