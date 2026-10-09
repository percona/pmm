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

import { fireEvent, render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { browserTimezone } from '@pmm-extensions/framework';
import { AutomationsPage } from '../src/AutomationsPage';
import type { OmGetBootstrapRunResponse } from '../src/types';

const { useOmBootstrapRuns } = vi.hoisted(() => ({
  useOmBootstrapRuns: vi.fn(),
}));

vi.mock('../src/inventoryHooks', () => ({ useOmBootstrapRuns }));
// The header is what these tests read; the Scans tab's content has its own suite.
vi.mock('../src/AutomationsScansTab', () => ({
  AutomationsScansTab: () => null,
}));

const NOW = '2026-10-06T12:00:00Z';
const RECENT = '2026-10-06T09:45:00Z';
const OLD = '2026-09-13T14:57:01Z';

const run = (
  run_id: string,
  started_at: string
): OmGetBootstrapRunResponse => ({
  run_id,
  status: 'succeeded',
  hosts: [],
  run_steps: [],
  replica_set_name: `rs-${run_id}`,
  mongodb_version: '8.0.4',
  started_at,
  finished_at: started_at.replace(':00Z', ':40Z'),
  cancel_requested: false,
});

const hover = (iso: string) =>
  `${new Date(iso).toLocaleString()} (${browserTimezone()})`;

describe('AutomationsPage installs', () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date(NOW));
    useOmBootstrapRuns.mockReturnValue({
      data: [run('recent', RECENT), run('old', OLD)],
      isLoading: false,
      error: null,
    });
    render(
      <MemoryRouter initialEntries={['/operations/automations']}>
        <AutomationsPage />
      </MemoryRouter>
    );
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it('shows an install from this week as relative time, with the zone on hover', () => {
    const cell = screen.getByTitle(hover(RECENT));
    expect(cell).toHaveTextContent('2 hours ago');
    expect(cell.closest('td')).toHaveStyle({ whiteSpace: 'nowrap' });
  });

  it('shows an older install as a local date and time, with the zone on hover', () => {
    expect(screen.getByTitle(hover(OLD))).toHaveTextContent(
      new Date(OLD).toLocaleString()
    );
  });

  it('never renders the old fixed yyyy-MM-dd HH:mm:ss format', () => {
    expect(
      screen.queryByText(/\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}/)
    ).toBeNull();
  });
});

describe('AutomationsPage header', () => {
  const NodesProbe = () => {
    const { pathname, search } = useLocation();
    return <output data-testid="location">{`${pathname}${search}`}</output>;
  };

  beforeEach(() => {
    useOmBootstrapRuns.mockReturnValue({
      data: [run('recent', RECENT)],
      isLoading: false,
      error: null,
    });
  });

  it.each(['installs', 'scans'])(
    'offers Install MongoDB as the primary action on the %s tab',
    (tab) => {
      render(
        <MemoryRouter initialEntries={[`/operations/automations?tab=${tab}`]}>
          <AutomationsPage />
        </MemoryRouter>
      );

      expect(screen.getByRole('link', { name: 'Install MongoDB' })).toHaveClass(
        'MuiButton-contained'
      );
    }
  );

  it('opens Nodes filtered to the nodes with nothing monitored on them', () => {
    render(
      <MemoryRouter initialEntries={['/operations/automations']}>
        <Routes>
          <Route path="/operations/automations" element={<AutomationsPage />} />
          <Route path="/operations/nodes" element={<NodesProbe />} />
        </Routes>
      </MemoryRouter>
    );

    fireEvent.click(screen.getByRole('link', { name: 'Install MongoDB' }));

    expect(screen.getByTestId('location')).toHaveTextContent(
      '/operations/nodes?filter=unmonitored'
    );
  });
});
