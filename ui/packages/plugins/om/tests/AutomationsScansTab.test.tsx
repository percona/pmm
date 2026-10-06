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

import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { browserTimezone } from '@pmm-extensions/framework';
import { AutomationsScansTab } from '../src/AutomationsScansTab';
import type { OmInventoryRun } from '../src/types';

const { useOmInventoryRuns, useIsEstateRefreshing, useRefreshInventory } =
  vi.hoisted(() => ({
    useOmInventoryRuns: vi.fn(),
    useIsEstateRefreshing: vi.fn(),
    useRefreshInventory: vi.fn(),
  }));

vi.mock('../src/inventoryHooks', () => ({
  useOmInventoryRuns,
  useIsEstateRefreshing,
  useRefreshInventory,
}));

const NOW = '2026-10-06T12:00:00Z';
const RECENT = '2026-10-06T09:00:00Z';
const OLD = '2026-09-20T08:30:00Z';

const run = (run_id: string, start_time: string): OmInventoryRun => ({
  run_id,
  status: 'RUN_STATUS_SUCCESS',
  start_time,
  end_time: start_time.replace(':00Z', ':39Z'),
  counts: {
    total_hosts: 1,
    probeable_hosts: 1,
    answered_hosts: 1,
    total_services: 1,
    resolved_services: 1,
    answered_services: 1,
    orphaned_services: 0,
  },
  scope: [],
});

const hover = (iso: string) =>
  `${new Date(iso).toLocaleString()} (${browserTimezone()})`;

describe('AutomationsScansTab', () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date(NOW));
    useOmInventoryRuns.mockReturnValue({
      data: [run('recent', RECENT), run('old', OLD)],
      isLoading: false,
      error: null,
    });
    useIsEstateRefreshing.mockReturnValue(false);
    useRefreshInventory.mockReturnValue({
      refreshAll: vi.fn(),
      isPending: false,
      error: null,
    });
    render(
      <MemoryRouter>
        <AutomationsScansTab />
      </MemoryRouter>
    );
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  const startedCell = (iso: string) =>
    screen
      .getAllByTitle(hover(iso))
      .find((el) => el.closest('td')) as HTMLElement;

  it('shows a run from this week as relative time, with the zone on hover', () => {
    const cell = startedCell(RECENT);
    expect(cell).toHaveTextContent('3 hours ago');
    expect(cell.closest('td')).toHaveStyle({ whiteSpace: 'nowrap' });
  });

  it('shows an older run as a local date and time, with the zone on hover', () => {
    expect(startedCell(OLD)).toHaveTextContent(new Date(OLD).toLocaleString());
  });

  it('never renders the old fixed yyyy-MM-dd HH:mm:ss format', () => {
    expect(
      screen.queryByText(/\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}/)
    ).toBeNull();
  });
});
