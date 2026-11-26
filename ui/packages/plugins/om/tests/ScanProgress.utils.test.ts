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
import { describeScanProgress } from '../src/components/ScanProgress.utils';
import type { OmInventoryRun } from '../src/types';

const NOW = Date.parse('2026-10-07T12:00:42Z');

const running = (probeable: number, finished: number): OmInventoryRun => ({
  run_id: 'run-1',
  status: 'RUN_STATUS_RUNNING',
  start_time: '2026-10-07T12:00:00Z',
  end_time: null,
  scope: [],
  counts: {
    total_hosts: probeable,
    probeable_hosts: probeable,
    answered_hosts: 0,
    finished_hosts: finished,
    total_services: 0,
    resolved_services: 0,
    answered_services: 0,
    orphaned_services: 0,
  },
});

describe('describeScanProgress', () => {
  it('says how many nodes are done, how long so far, and how long one usually takes', () => {
    expect(describeScanProgress(running(12, 3), 60, NOW)).toBe(
      '3 of 12 nodes done · 42s so far · usually about 1m'
    );
  });

  it('leaves the node count out when there is no node to scan', () => {
    expect(describeScanProgress(running(0, 0), null, NOW)).toBe('42s so far');
  });

  it('leaves the usual time out when no scan like it has finished', () => {
    expect(describeScanProgress(running(1, 0), null, NOW)).toBe(
      '0 of 1 node done · 42s so far'
    );
  });

  it('never counts the time so far below zero', () => {
    expect(
      describeScanProgress(
        running(0, 0),
        null,
        Date.parse('2026-10-07T11:59:00Z')
      )
    ).toBe('0s so far');
  });
});
