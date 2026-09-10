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
import { describe, expect, it, vi } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import { SnackbarProvider } from 'notistack';
import type { PluginSchema } from '@sep/api';
import { PluginListPage } from './PluginListPage';

vi.mock('../SchemaListView', () => ({
  SchemaListView: () => <div>list</div>,
}));

vi.mock('@sep/api', () => ({
  DEFAULT_PLUGIN_LIST_OFFSET: 0,
  DEFAULT_PLUGIN_LIST_LIMIT: 50,
  RUNNING_STATUSES: new Set(['running', 'pending']),
  usePluginTasks: () => ({
    data: { items: [], pagination: null },
    isLoading: false,
  }),
  usePluginEntityList: () => ({
    data: { items: [], pagination: null },
    isLoading: false,
  }),
  useDeletePluginEntity: () => ({
    mutate: vi.fn(),
    isPending: false,
    variables: undefined,
  }),
}));

const schema: PluginSchema = {
  name: 'sched',
  display_name: 'Sched',
  capabilities: { scheduling: true },
  list_view: { columns: [{ key: 'name', label: 'Name' }] },
};

function renderPage(props: Partial<Parameters<typeof PluginListPage>[0]> = {}) {
  return render(
    <SnackbarProvider>
      <MemoryRouter>
        <PluginListPage schema={schema} pluginName="sched" {...props} />
      </MemoryRouter>
    </SnackbarProvider>
  );
}

describe('PluginListPage — generic Schedules button', () => {
  it('renders the generic Schedules button by default when scheduling is enabled', () => {
    renderPage();
    expect(screen.getByTestId('plugin-schedule-link')).toBeInTheDocument();
  });

  it('suppresses the generic Schedules button when hideScheduleButton is set', () => {
    renderPage({ hideScheduleButton: true });
    expect(
      screen.queryByTestId('plugin-schedule-link')
    ).not.toBeInTheDocument();
  });
});
