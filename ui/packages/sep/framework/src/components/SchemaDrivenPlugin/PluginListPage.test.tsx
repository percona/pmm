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

import { act, render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { ReactNode } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { SnackbarProvider } from 'notistack';
import type { PluginSchema } from '@sep/api';
import { PluginListPage } from './PluginListPage';

const { schemaListViewMock, deleteEntityMock, authMock } = vi.hoisted(() => ({
  schemaListViewMock: vi.fn(),
  deleteEntityMock: vi.fn(),
  /** Flipped per test to cover the read-only (non-admin) rendering. */
  authMock: { canMutate: true },
}));

vi.mock('../SchemaListView', () => ({
  SchemaListView: (props: Record<string, unknown>) => {
    schemaListViewMock(props);
    return <div>list</div>;
  },
}));

vi.mock('@sep/api', () => ({
  useAuth: () => ({
    isAdmin: authMock.canMutate,
    canMutate: authMock.canMutate,
  }),
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
    mutate: deleteEntityMock,
    isPending: false,
    variables: undefined,
  }),
  ApiError: class ApiError extends Error {
    status?: number;
    data?: unknown;
    constructor(details: { status?: number; message: string; data?: unknown }) {
      super(details.message);
      this.status = details.status;
      this.data = details.data;
    }
  },
  parseFieldErrors: (error: { data?: { detail?: unknown } }) =>
    Array.isArray(error?.data?.detail)
      ? (error.data.detail as { loc?: string[]; msg?: string }[]).map(
          (entry) => ({
            path: (entry.loc ?? []).filter((seg) => seg !== 'body').join('.'),
            message: entry.msg ?? 'Invalid value',
          })
        )
      : [],
}));

beforeEach(() => {
  schemaListViewMock.mockClear();
  deleteEntityMock.mockReset();
  authMock.canMutate = true;
});

const schema: PluginSchema = {
  name: 'sched',
  display_name: 'Sched',
  item_display_name: 'sched',
  item_display_name_plural: 'scheds',
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

describe('PluginListPage — status column opens the last run', () => {
  const renderWithClient = (ui: ReactNode) =>
    render(
      <QueryClientProvider
        client={
          new QueryClient({
            defaultOptions: { queries: { retry: false, gcTime: 0 } },
          })
        }
      >
        <SnackbarProvider>
          <MemoryRouter>{ui}</MemoryRouter>
        </SnackbarProvider>
      </QueryClientProvider>
    );

  const statusSchema: PluginSchema = {
    name: 'sched',
    display_name: 'Sched',
    list_view: {
      columns: [
        { key: 'name', label: 'Name' },
        { key: 'status', label: 'Status', format: 'status' },
      ],
    },
  } as unknown as PluginSchema;

  const statusCell = (row: Record<string, unknown> = { name: 'backup' }) => {
    const render = schemaListViewMock.mock.calls.at(-1)?.[0]
      .renderListColumn as (context: {
      columnKey: string;
      value: unknown;
      row: Record<string, unknown>;
    }) => ReactNode | undefined;

    return render({ columnKey: 'status', value: 'failed', row });
  };

  it('opens the run without also following the row', async () => {
    // The row navigates to the task detail. Both handlers see the same native
    // click, so without a stopPropagation one click would open the run and
    // leave the page at the same time.
    renderWithClient(
      <PluginListPage schema={statusSchema} pluginName="sched" />
    );

    const onRowClick = vi.fn();
    render(<div onClick={onRowClick}>{statusCell()}</div>);

    await userEvent.click(screen.getByRole('button', { name: /Failed/ }));

    expect(onRowClick).not.toHaveBeenCalled();
  });

  it('leaves the chip inert on a multi-entity list', () => {
    // A row there is an entity, so `name` is not a task name and there is no
    // run to resolve from it.
    render(
      <SnackbarProvider>
        <MemoryRouter initialEntries={['/apps/sched/inventory']}>
          <Routes>
            <Route
              path="/apps/:plugin/:entityName"
              element={
                <PluginListPage
                  schema={
                    {
                      ...statusSchema,
                      entities: [
                        {
                          name: 'inventory',
                          display_name: 'Inventory',
                          list_view: statusSchema.list_view,
                        },
                      ],
                    } as unknown as PluginSchema
                  }
                  pluginName="sched"
                />
              }
            />
          </Routes>
        </MemoryRouter>
      </SnackbarProvider>
    );

    expect(statusCell()).toBeUndefined();
  });

  it('defers to a plugin-supplied override for the same column', () => {
    const renderListColumn = vi.fn(() => <span>custom</span>);
    renderWithClient(
      <PluginListPage
        schema={statusSchema}
        pluginName="sched"
        renderListColumn={renderListColumn}
      />
    );

    render(<div>{statusCell()}</div>);

    expect(screen.getByText('custom')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /Failed/ })).toBeNull();
  });
});

describe('PluginListPage — write access', () => {
  const deletableEntitySchema = {
    name: 'inventory',
    display_name: 'Inventory',
    capabilities: { scheduling: false },
    entities: [
      {
        name: 'nodes',
        display_name: 'Nodes',
        forms: [],
        list_view: {
          columns: [
            { key: 'name', label: 'Name' },
            { key: '_actions', label: 'Actions', format: 'actions' },
          ],
        },
      },
    ],
  } as unknown as PluginSchema;

  function renderDeletableEntityList() {
    return render(
      <SnackbarProvider>
        <MemoryRouter initialEntries={['/inventory/nodes']}>
          <Routes>
            <Route
              path="/:plugin/:entityName"
              element={
                <PluginListPage
                  schema={deletableEntitySchema}
                  pluginName="inventory"
                  allowListEntityDelete
                />
              }
            />
          </Routes>
        </MemoryRouter>
      </SnackbarProvider>
    );
  }

  function lastListViewProps() {
    return schemaListViewMock.mock.calls.at(-1)?.[0] as {
      onDeleteRow?: unknown;
    };
  }

  it('renders the create button and wires row delete for a session that may mutate', () => {
    renderPage();
    expect(
      screen.getByRole('button', { name: 'New Sched' })
    ).toBeInTheDocument();

    renderDeletableEntityList();
    expect(lastListViewProps().onDeleteRow).toBeInstanceOf(Function);
  });

  it('renders no create button and no row delete for a non-admin', () => {
    authMock.canMutate = false;

    renderPage();
    expect(
      screen.queryByRole('button', { name: 'New Sched' })
    ).not.toBeInTheDocument();
    // Reads stay: the list itself and the Schedules link are unaffected.
    expect(screen.getByTestId('plugin-schedule-link')).toBeInTheDocument();

    renderDeletableEntityList();
    expect(lastListViewProps().onDeleteRow).toBeUndefined();
  });
});

describe('PluginListPage — delete failure reporting', () => {
  const deletableEntitySchema = {
    name: 'inventory',
    display_name: 'Inventory',
    capabilities: { scheduling: false },
    entities: [
      {
        name: 'nodes',
        display_name: 'Nodes',
        forms: [],
        list_view: {
          columns: [
            { key: 'name', label: 'Name' },
            { key: '_actions', label: 'Actions', format: 'actions' },
          ],
        },
      },
    ],
  } as unknown as PluginSchema;

  function renderDeletableEntityList() {
    return render(
      <SnackbarProvider>
        <MemoryRouter initialEntries={['/inventory/nodes']}>
          <Routes>
            <Route
              path="/:plugin/:entityName"
              element={
                <PluginListPage
                  schema={deletableEntitySchema}
                  pluginName="inventory"
                  allowListEntityDelete
                />
              }
            />
          </Routes>
        </MemoryRouter>
      </SnackbarProvider>
    );
  }

  async function confirmDelete() {
    const props = schemaListViewMock.mock.calls.at(-1)?.[0] as {
      onDeleteRow?: (row: Record<string, unknown>) => void;
    };
    act(() => props.onDeleteRow?.({ id: 1, name: 'node-a' }));
    const dialog = await screen.findByRole('dialog');
    await userEvent.click(
      within(dialog).getByRole('button', { name: 'Delete' })
    );
  }

  it("reports a refused delete on the list with the server's own reason", async () => {
    deleteEntityMock.mockImplementation((_id, opts) =>
      opts.onError?.(
        new Error("You don't have permission to perform this action")
      )
    );
    renderDeletableEntityList();

    await confirmDelete();

    expect(
      await screen.findByTestId('plugin-list-action-error')
    ).toHaveTextContent("You don't have permission to perform this action");
    // The alert replaces the previous error toast rather than joining it.
    expect(document.querySelector('[class*="notistack"]')).toBeNull();
  });

  it('reports nothing when the delete succeeds', async () => {
    deleteEntityMock.mockImplementation((_id, opts) => opts.onSuccess?.());
    renderDeletableEntityList();

    await confirmDelete();

    await waitFor(() => expect(deleteEntityMock).toHaveBeenCalledTimes(1));
    expect(
      screen.queryByTestId('plugin-list-action-error')
    ).not.toBeInTheDocument();
  });
});
