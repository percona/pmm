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
 * The Clusters tab's search and filters, as URL parameters and as a row predicate.
 *
 * In the URL rather than component state, so a filtered view survives being pasted to
 * a colleague and Back walks through it (PMM-15806). Group by is deliberately not here:
 * a shareable filter matters, a shareable grouping does not (PMM-15767).
 */

import type { OmClusterHealth, OmClusterRow, OmClusterType } from './types';

/** Worst first, the order the health filter offers them in. */
export const HEALTH_FILTER_VALUES: OmClusterHealth[] = [
  'down',
  'degraded',
  'unknown',
  'healthy',
];

/** How each cluster type is spelled in the URL. */
export const TYPE_FILTER_VALUES: Record<string, OmClusterType> = {
  'replica-set': 'CLUSTER_TYPE_REPLICA_SET',
  sharded: 'CLUSTER_TYPE_SHARDED',
  standalone: 'CLUSTER_TYPE_STANDALONE',
};

export interface ClusterFilters {
  /** Case-insensitive substring of the cluster name; empty is no search. */
  q: string;
  health: OmClusterHealth[];
  /** URL spellings, keys of {@link TYPE_FILTER_VALUES}. */
  type: string[];
  /** Environment names; the empty string is the unnamed environment. */
  environment: string[];
}

const FILTER_KEYS = ['q', 'health', 'type', 'environment'] as const;

const unique = (values: string[]) => [...new Set(values)];

/**
 * Read the filters off the URL. A value the filter does not offer is dropped, so a
 * link from an older or newer PMM opens on what this one can show.
 */
export function parseClusterFilters(params: URLSearchParams): ClusterFilters {
  return {
    // Untrimmed: the search box reads it back, and must keep a space being typed.
    q: params.get('q') ?? '',
    health: unique(params.getAll('health')).filter((value) =>
      HEALTH_FILTER_VALUES.includes(value as OmClusterHealth)
    ) as OmClusterHealth[],
    type: unique(params.getAll('type')).filter((value) =>
      // Not `in`: that admits inherited names such as `toString`. Not `Object.hasOwn`
      // either, which the PMM app's TypeScript lib predates.
      Object.prototype.hasOwnProperty.call(TYPE_FILTER_VALUES, value)
    ),
    environment: unique(params.getAll('environment')),
  };
}

/** Write the filters into a copy of `params`, leaving out defaults and keeping the rest. */
export function withClusterFilters(
  params: URLSearchParams,
  filters: ClusterFilters
): URLSearchParams {
  const next = new URLSearchParams(params);
  for (const key of FILTER_KEYS) {
    next.delete(key);
  }
  if (filters.q.trim()) {
    next.set('q', filters.q);
  }
  for (const key of ['health', 'type', 'environment'] as const) {
    for (const value of filters[key]) {
      next.append(key, value);
    }
  }
  return next;
}

export const hasClusterFilters = (filters: ClusterFilters) =>
  !!filters.q.trim() ||
  filters.health.length > 0 ||
  filters.type.length > 0 ||
  filters.environment.length > 0;

/**
 * The rows matching every filter, and any value within one.
 *
 * An environment the fleet no longer has is ignored rather than matching nothing, the
 * same treatment as an unknown health or type: a stale link opens on the fleet, not on
 * an empty table that cannot say why.
 */
export function filterClusterRows(
  rows: OmClusterRow[],
  filters: ClusterFilters
): OmClusterRow[] {
  const query = filters.q.trim().toLowerCase();
  const present = new Set(rows.map((row) => row.env_name ?? ''));
  const environments = filters.environment.filter((name) => present.has(name));
  const types = filters.type.map((value) => TYPE_FILTER_VALUES[value]);
  return rows.filter(
    (row) =>
      (!query || (row.cluster_name ?? '').toLowerCase().includes(query)) &&
      (!filters.health.length || filters.health.includes(row.health)) &&
      (!types.length || types.includes(row.type)) &&
      (!environments.length || environments.includes(row.env_name ?? ''))
  );
}
