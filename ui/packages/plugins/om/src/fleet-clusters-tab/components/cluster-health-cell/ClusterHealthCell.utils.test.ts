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
import { cluster, service, topology } from '../../../../tests/fixtures';
import { toClusterRows } from '../../../topology';
import type { OmService } from '../../../types';
import { healthReason } from './ClusterHealthCell.utils';

const DOWN = 'SERVICE_STATUS_DOWN';
const SILENT = 'SERVICE_STATUS_UNSPECIFIED';

const rowOf = (statuses: OmService['status'][]) =>
  toClusterRows(
    topology([
      {
        env_name: 'production',
        clusters: [
          cluster({
            name: 'c',
            services: statuses.map((status, index) =>
              service({ service_name: `host-${index + 1}`, status })
            ),
          }),
        ],
      },
    ])
  )[0];

describe('healthReason', () => {
  it('says nothing for a healthy cluster', () => {
    expect(healthReason(rowOf(['SERVICE_STATUS_UP']))).toBeUndefined();
  });

  it('counts every member of a down cluster', () => {
    expect(healthReason(rowOf([DOWN, DOWN, DOWN]))).toEqual({
      text: 'All 3 members down',
    });
    expect(healthReason(rowOf([DOWN]))).toEqual({
      text: 'Its only member is down',
    });
  });

  it('names the one down member of a degraded cluster', () => {
    expect(
      healthReason(rowOf(['SERVICE_STATUS_UP', DOWN, 'SERVICE_STATUS_UP']))
    ).toEqual({ text: 'host-2 down' });
  });

  it('counts several down members, listing them for the tooltip', () => {
    expect(
      healthReason(
        rowOf([DOWN, 'SERVICE_STATUS_UP', DOWN, 'SERVICE_STATUS_UP'])
      )
    ).toEqual({ text: '2 of 4 members down', members: ['host-1', 'host-3'] });
  });

  it('counts the silent members of an unknown cluster, or says it has none', () => {
    expect(healthReason(rowOf(['SERVICE_STATUS_UP', SILENT]))).toEqual({
      text: '1 of 2 members not reporting',
      members: ['host-2'],
    });
    expect(healthReason(rowOf([]))).toEqual({
      text: 'No members in this snapshot',
    });
  });
});
