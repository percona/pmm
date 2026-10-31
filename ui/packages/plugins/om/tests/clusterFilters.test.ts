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

import { describe, expect, it } from 'vitest';
import {
  filterClusterRows,
  parseClusterFilters,
  withClusterFilters,
  type ClusterFilters,
} from '../src/clusterFilters';
import { toClusterRows } from '../src/topology';
import { cluster, service, topology } from './fixtures';

const filters = (overrides: Partial<ClusterFilters>): ClusterFilters => ({
  q: '',
  health: [],
  type: [],
  environment: [],
  ...overrides,
});

const down = { status: 'SERVICE_STATUS_DOWN' } as const;

const rows = toClusterRows(
  topology([
    {
      env_name: 'production',
      clusters: [
        cluster({
          name: 'Orders-RS',
          services: [service({ service_name: 'o1' })],
        }),
        cluster({
          name: 'billing',
          type: 'CLUSTER_TYPE_SHARDED',
          services: [service({ service_name: 'b1', ...down })],
        }),
      ],
    },
    {
      env_name: null,
      clusters: [
        cluster({
          name: 'orders-legacy',
          type: 'CLUSTER_TYPE_STANDALONE',
          services: [
            service({ service_name: 'l1' }),
            service({ service_name: 'l2', ...down }),
          ],
        }),
      ],
    },
  ])
);

const names = (matched: typeof rows) => matched.map((row) => row.cluster_name);

describe('parseClusterFilters', () => {
  it('reads every filter, several values each', () => {
    expect(
      parseClusterFilters(
        new URLSearchParams(
          'tab=clusters&q=ord&health=down&health=degraded&type=sharded&environment=production&environment='
        )
      )
    ).toEqual({
      q: 'ord',
      health: ['down', 'degraded'],
      type: ['sharded'],
      environment: ['production', ''],
    });
  });

  it('drops a health or type it does not offer, and repeats', () => {
    expect(
      parseClusterFilters(
        new URLSearchParams('health=on-fire&health=down&health=down&type=mesh')
      )
    ).toEqual(filters({ health: ['down'] }));
  });

  it('drops a type named like an inherited property', () => {
    expect(
      parseClusterFilters(
        new URLSearchParams('type=toString&type=constructor&type=__proto__')
      ).type
    ).toEqual([]);
  });
});

describe('withClusterFilters', () => {
  it('keeps the other parameters and leaves out defaults', () => {
    const next = withClusterFilters(
      new URLSearchParams('tab=clusters&q=old&health=down'),
      filters({ type: ['replica-set'], q: '  ' })
    );

    expect(next.toString()).toBe('tab=clusters&type=replica-set');
  });

  it('round-trips through the parser', () => {
    const written = filters({
      q: 'ord',
      health: ['healthy', 'unknown'],
      type: ['standalone'],
      environment: [''],
    });

    expect(
      parseClusterFilters(
        withClusterFilters(new URLSearchParams('tab=clusters'), written)
      )
    ).toEqual(written);
  });
});

describe('filterClusterRows', () => {
  it('matches the cluster name case-insensitively', () => {
    expect(names(filterClusterRows(rows, filters({ q: ' ORDERS ' })))).toEqual([
      'Orders-RS',
      'orders-legacy',
    ]);
  });

  it('ORs the values of one filter', () => {
    expect(
      names(filterClusterRows(rows, filters({ health: ['down', 'degraded'] })))
    ).toEqual(['billing', 'orders-legacy']);
  });

  it('filters by type', () => {
    expect(
      names(
        filterClusterRows(rows, filters({ type: ['sharded', 'standalone'] }))
      )
    ).toEqual(['billing', 'orders-legacy']);
  });

  it('filters by environment, the unnamed one included', () => {
    expect(
      names(filterClusterRows(rows, filters({ environment: [''] })))
    ).toEqual(['orders-legacy']);
  });

  it('ANDs filters and search', () => {
    expect(
      names(
        filterClusterRows(
          rows,
          filters({ q: 'orders', environment: ['production'] })
        )
      )
    ).toEqual(['Orders-RS']);
  });

  it('ignores an environment the fleet does not have', () => {
    expect(
      filterClusterRows(rows, filters({ environment: ['staging'] }))
    ).toHaveLength(3);
  });
});
