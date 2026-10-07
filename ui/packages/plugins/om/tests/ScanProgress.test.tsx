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

import { act, render, screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ScanProgress } from '../src/components/ScanProgress';
import type { OmInventoryRun } from '../src/types';

const { useScanInFlight } = vi.hoisted(() => ({ useScanInFlight: vi.fn() }));

vi.mock('../src/inventoryHooks', () => ({ useScanInFlight }));

const NOW = Date.parse('2026-10-07T12:00:42Z');

const running = (finished?: number): OmInventoryRun => ({
  run_id: 'run-1',
  status: 'RUN_STATUS_RUNNING',
  start_time: '2026-10-07T12:00:00Z',
  end_time: null,
  scope: [],
  counts: {
    total_hosts: 12,
    probeable_hosts: 12,
    answered_hosts: 0,
    finished_hosts: finished,
    total_services: 0,
    resolved_services: 0,
    answered_services: 0,
    orphaned_services: 0,
  },
});

describe('ScanProgress', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(NOW);
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('says how many nodes are done, how long so far, and how long one usually takes', () => {
    useScanInFlight.mockReturnValue({ run: running(3), expectedSeconds: 60 });

    render(<ScanProgress />);

    expect(screen.getByTestId('om-scan-progress')).toHaveTextContent(
      '3 of 12 nodes done · 42s so far · usually about 1m'
    );
  });

  it('keeps counting the time while the scan runs', () => {
    useScanInFlight.mockReturnValue({ run: running(3), expectedSeconds: null });
    render(<ScanProgress />);

    act(() => {
      vi.advanceTimersByTime(2000);
    });

    expect(screen.getByTestId('om-scan-progress')).toHaveTextContent(
      '3 of 12 nodes done · 44s so far'
    );
  });

  it('shows the time alone against a server that does not count nodes', () => {
    useScanInFlight.mockReturnValue({
      run: running(undefined),
      expectedSeconds: null,
    });

    render(<ScanProgress />);

    expect(screen.getByTestId('om-scan-progress')).toHaveTextContent(
      /^42s so far$/
    );
  });

  it('shows nothing while no scan runs', () => {
    useScanInFlight.mockReturnValue({ run: undefined, expectedSeconds: null });

    render(<ScanProgress />);

    expect(screen.queryByTestId('om-scan-progress')).toBeNull();
  });
});
