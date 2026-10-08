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

import { render, screen, within } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { browserTimezone } from '@pmm-extensions/framework';
import { AutomationsScansTab } from '../src/AutomationsScansTab';
import type { OmInventoryRun } from '../src/types';

const {
  useOmInventoryRuns,
  useOmInventoryRun,
  useRefreshInventory,
  useIsEstateRefreshing,
} = vi.hoisted(() => ({
  useOmInventoryRuns: vi.fn(),
  useOmInventoryRun: vi.fn(),
  useRefreshInventory: vi.fn(),
  useIsEstateRefreshing: vi.fn(),
}));

vi.mock('../src/inventoryHooks', () => ({
  useOmInventoryRuns,
  useOmInventoryRun,
  useRefreshInventory,
  useIsEstateRefreshing,
  useActiveInventoryRun: () => ({ run: undefined, updatedAt: 0 }),
}));

const COUNTS: OmInventoryRun['counts'] = {
  total_hosts: 1,
  probeable_hosts: 1,
  answered_hosts: 0,
  total_services: 0,
  resolved_services: 0,
  answered_services: 0,
  orphaned_services: 0,
};

const run = (run_id: string): OmInventoryRun => ({
  run_id,
  status: 'RUN_STATUS_FAILED',
  start_time: new Date().toISOString(),
  end_time: new Date().toISOString(),
  scope: [],
  counts: COUNTS,
});

const renderAt = (url: string) =>
  render(
    <MemoryRouter initialEntries={[url]}>
      <AutomationsScansTab />
    </MemoryRouter>
  );

describe('AutomationsScansTab', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useOmInventoryRuns.mockReturnValue({
      data: [run('run-1'), run('run-42')],
      isLoading: false,
      error: null,
    });
    useOmInventoryRun.mockReturnValue({ isPending: true });
    useRefreshInventory.mockReturnValue({
      refreshAll: vi.fn(),
      isPending: false,
      error: null,
    });
    useIsEstateRefreshing.mockReturnValue(false);
  });

  // The Nodes page links a failing node's scan here; landing has to open that scan,
  // not leave the reader to find it in the list. A run's panel fetches its own
  // detail, so which run it was asked for is which row is open.
  it('opens the run named by ?expand on landing', () => {
    renderAt('/automations?tab=scans&expand=run-42');

    expect(useOmInventoryRun).toHaveBeenCalledWith('run-42');
    expect(useOmInventoryRun).not.toHaveBeenCalledWith('run-1');
  });

  it('opens nothing without it', () => {
    renderAt('/automations?tab=scans');

    expect(useOmInventoryRun).not.toHaveBeenCalled();
  });

  describe('timestamps', () => {
    const NOW = '2026-10-06T12:00:00Z';
    const RECENT = '2026-10-06T09:00:00Z';
    const OLD = '2026-09-20T08:30:00Z';

    const finishedRun = (
      run_id: string,
      start_time: string
    ): OmInventoryRun => ({
      run_id,
      status: 'RUN_STATUS_SUCCESS',
      start_time,
      end_time: start_time.replace(':00Z', ':39Z'),
      counts: {
        ...COUNTS,
        answered_hosts: 1,
        total_services: 1,
        resolved_services: 1,
        answered_services: 1,
      },
      scope: [],
    });

    const hover = (iso: string) =>
      `${new Date(iso).toLocaleString()} (${browserTimezone()})`;

    beforeEach(() => {
      vi.useFakeTimers({ toFake: ['Date'] });
      vi.setSystemTime(new Date(NOW));
      useOmInventoryRuns.mockReturnValue({
        // Different outcomes, so the two stay separate rows rather than one group.
        data: [
          finishedRun('recent', RECENT),
          { ...finishedRun('old', OLD), status: 'RUN_STATUS_FAILED' },
        ],
        isLoading: false,
        error: null,
      });
      renderAt('/automations?tab=scans');
    });
    afterEach(() => {
      vi.useRealTimers();
    });

    // The last-scan summary carries the same hover, so pick the one inside the table.
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
      expect(startedCell(OLD)).toHaveTextContent(
        new Date(OLD).toLocaleString()
      );
    });

    it('never renders the old fixed yyyy-MM-dd HH:mm:ss format', () => {
      expect(
        screen.queryByText(/\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}/)
      ).toBeNull();
    });

    it('gives the last scan summary its age and a hover', () => {
      expect(screen.getByText('3h ago')).toHaveAttribute(
        'title',
        hover(RECENT)
      );
    });
  });

  describe('grouped history', () => {
    const at = (minutesAgo: number, run_id: string): OmInventoryRun => ({
      ...run(run_id),
      status: 'RUN_STATUS_PARTIAL',
      start_time: new Date(Date.now() - minutesAgo * 60_000).toISOString(),
      failing_nodes: [{ node_id: 'n2', name: 'node02' }],
    });

    beforeEach(() => {
      useOmInventoryRuns.mockReturnValue({
        data: [at(0, 'run-3'), at(10, 'run-2'), at(20, 'run-1')],
        isLoading: false,
        error: null,
      });
    });

    it('shows three scans that failed the same way as one row, with how long', () => {
      renderAt('/automations?tab=scans');

      expect(screen.getAllByTestId('om-run-group-span')).toHaveLength(1);
      expect(screen.getByTestId('om-run-group-span')).toHaveTextContent(
        /^3 scans over 20m( \d+s)?$/
      );
    });

    it('links each failing node to its row on the Nodes page', () => {
      renderAt('/automations?tab=scans');

      // The newest-run summary above the table names none; the table row does.
      const link = screen.getByRole('link', { name: 'node02' });
      expect(link).toHaveAttribute(
        'href',
        expect.stringContaining('nodes?node=node02')
      );
    });

    it('opens the group holding the run ?expand names, and that run in it', () => {
      renderAt('/automations?tab=scans&expand=run-2');

      expect(
        within(screen.getByTestId('om-run-group')).getAllByRole('button', {
          expanded: true,
        })
      ).toHaveLength(1);
      expect(useOmInventoryRun).toHaveBeenCalledWith('run-2');
      expect(useOmInventoryRun).not.toHaveBeenCalledWith('run-3');
    });
  });
});
