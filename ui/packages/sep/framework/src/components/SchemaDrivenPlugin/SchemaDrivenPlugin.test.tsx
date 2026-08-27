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

import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { SnackbarProvider } from 'notistack';
import { type PluginSchema } from '@sep/api';
import { SchemaDrivenPlugin } from './SchemaDrivenPlugin';
import type { RenderFormSlot } from './types';

const mockUpdateMutate = vi.fn();

const schema: PluginSchema = {
  pluginName: 'inventory',
  display_name: 'Inventory',
  entities: [
    {
      name: 'nodes',
      display_name: 'Nodes',
      forms: [
        {
          title: 'Main',
          fields: [{ type: 'string', name: 'label', label: 'Label' }],
        },
      ],
      list_view: { columns: [{ key: 'label', label: 'Label' }] },
    },
  ],
} as unknown as PluginSchema;

// Single-entity (task-style) schema for the task/:id/edit branch.
const taskSchema: PluginSchema = {
  pluginName: 'checksums',
  display_name: 'Checksum',
  forms: [
    {
      title: 'Main',
      fields: [
        { type: 'string', name: 'task_name', label: 'Task Name' },
        { type: 'string', name: 'title', label: 'Title' },
      ],
    },
  ],
  list_view: { columns: [{ key: 'name', label: 'Name' }] },
} as unknown as PluginSchema;

const backupsSchema: PluginSchema = {
  name: 'mysql_backups',
  display_name: 'MySQL Backups',
  forms: [
    {
      title: 'Main',
      fields: [{ type: 'string', name: 'name', label: 'Name' }],
    },
  ],
  list_view: { columns: [{ key: 'name', label: 'Name' }] },
  related_apps: [
    {
      app_key: 'mysql_backups/restore',
      label: 'Restore',
      route_segment: 'restores',
    },
  ],
} as unknown as PluginSchema;

const restoreSchema: PluginSchema = {
  name: 'mysql_backups_restore',
  display_name: 'Restore',
  forms: [
    {
      title: 'Main',
      fields: [{ type: 'string', name: 'name', label: 'Name' }],
    },
  ],
  list_view: { columns: [{ key: 'name', label: 'Name' }] },
} as unknown as PluginSchema;

const taskRecord = {
  id: 1,
  name: 'check1',
  data: { _form: { task_name: 'check1', title: 'hello' } },
};

// Schema the mocked usePluginSchema serves; per-test override, reset after each.
let activeSchema: PluginSchema = schema;

// Stub sibling page modules so their @sep/api imports stay out of the graph;
// this test exercises only the SchemaDrivenPlugin → edit-page threading.
vi.mock('./PluginListPage', () => ({
  PluginListPage: ({ pluginName }: { pluginName: string }) => (
    <div>list:{pluginName}</div>
  ),
}));
vi.mock('./PluginDetailPage', () => ({
  PluginDetailPage: () => <div>detail</div>,
  pathToEntityList: () => '',
}));
vi.mock('./PluginSchedulePage', () => ({
  PluginSchedulePage: () => <div>schedule</div>,
}));

/** Flipped per test to cover the read-only (non-admin) rendering. */
let mockCanMutate = true;

vi.mock('@sep/api', () => ({
  useAuth: () => ({ isAdmin: mockCanMutate, canMutate: mockCanMutate }),
  usePluginSchema: (pluginName: string) => {
    if (pluginName === 'mysql_backups/restore') {
      return { data: restoreSchema, isLoading: false, error: null };
    }
    return { data: activeSchema, isLoading: false, error: null };
  },
  usePluginEntityDetail: () => ({
    data: { id: 5, label: 'n1' },
    isLoading: false,
  }),
  useUpdatePluginEntity: () => ({ mutate: mockUpdateMutate, isPending: false }),
  useCreatePluginEntity: () => ({ mutate: vi.fn(), isPending: false }),
  useCreatePluginTask: () => ({ mutate: vi.fn(), isPending: false }),
  usePluginTask: () => ({ data: taskRecord, isLoading: false }),
  useUpdatePluginTask: () => ({
    mutate: vi.fn(),
    isPending: false,
    isError: false,
    error: null,
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

afterEach(() => {
  activeSchema = schema;
  mockCanMutate = true;
  mockUpdateMutate.mockReset();
});

function renderEdit(renderEditForm?: RenderFormSlot) {
  return render(
    <SnackbarProvider>
      <MemoryRouter initialEntries={['/nodes/5/edit']}>
        <SchemaDrivenPlugin
          pluginName="inventory"
          renderEditForm={renderEditForm}
        />
      </MemoryRouter>
    </SnackbarProvider>
  );
}

describe('SchemaDrivenPlugin — renderEditForm slot', () => {
  it('renders the default form body when no slot is supplied', () => {
    renderEdit();
    expect(screen.getByText('Edit Nodes #5')).toBeInTheDocument();
    expect(screen.getByLabelText('Label')).toBeInTheDocument();
    expect(screen.queryByTestId('custom-edit-form')).toBeNull();
  });

  it('replaces only the form body with the slot, keeping chrome and mutation wiring', async () => {
    const user = userEvent.setup();
    const renderEditForm: RenderFormSlot = ({ onSubmit, defaultValues }) => (
      <button
        type="button"
        onClick={() => onSubmit({ ...defaultValues, label: 'edited' })}
      >
        Save slot
      </button>
    );
    renderEdit(renderEditForm);

    // Chrome preserved; default form body replaced by the slot.
    expect(screen.getByText('Edit Nodes #5')).toBeInTheDocument();
    expect(screen.queryByLabelText('Label')).toBeNull();

    await user.click(screen.getByRole('button', { name: 'Save slot' }));
    await waitFor(() => expect(mockUpdateMutate).toHaveBeenCalledTimes(1));
    expect(mockUpdateMutate).toHaveBeenCalledWith(
      { id: '5', values: expect.objectContaining({ label: 'edited' }) },
      expect.objectContaining({
        onSuccess: expect.any(Function),
        onError: expect.any(Function),
      })
    );
  });
});

describe('SchemaDrivenPlugin — single-entity task edit route', () => {
  it('renders PluginTaskEditPage at task/:id/edit, prefilled from the stored form', () => {
    activeSchema = taskSchema;
    render(
      <SnackbarProvider>
        <MemoryRouter initialEntries={['/task/check1/edit']}>
          <SchemaDrivenPlugin pluginName="checksums" />
        </MemoryRouter>
      </SnackbarProvider>
    );

    expect(screen.getByText('Edit Checksum: check1')).toBeInTheDocument();
    expect(screen.getByLabelText('Title')).toHaveValue('hello');
    // task_name stays immutable: no editable name input is rendered.
    expect(screen.queryByLabelText('Task Name')).toBeNull();
  });
});

describe('SchemaDrivenPlugin — related_apps routing', () => {
  function renderBackupsPlugin(pathname: string) {
    return render(
      <SnackbarProvider>
        <MemoryRouter initialEntries={[pathname]}>
          <Routes>
            <Route
              path="/apps/mysql_backups/*"
              element={
                <SchemaDrivenPlugin
                  pluginName="mysql_backups"
                  routeBase="/apps/mysql_backups"
                />
              }
            />
          </Routes>
        </MemoryRouter>
      </SnackbarProvider>
    );
  }

  it('does not render the related-app tab bar when related_apps is absent', () => {
    activeSchema = taskSchema;
    render(
      <SnackbarProvider>
        <MemoryRouter initialEntries={['/']}>
          <SchemaDrivenPlugin
            pluginName="checksums"
            routeBase="/apps/checksums"
          />
        </MemoryRouter>
      </SnackbarProvider>
    );

    expect(screen.getByText('list:checksums')).toBeInTheDocument();
    expect(screen.queryByRole('tablist')).toBeNull();
  });

  it('renders the tab bar and parent list on the parent route', () => {
    activeSchema = backupsSchema;
    renderBackupsPlugin('/apps/mysql_backups');

    expect(screen.getByRole('tab', { name: 'MySQL Backups' })).toHaveAttribute(
      'aria-selected',
      'true'
    );
    expect(screen.getByRole('tab', { name: 'Restore' })).toHaveAttribute(
      'aria-selected',
      'false'
    );
    expect(screen.getByText('list:mysql_backups')).toBeInTheDocument();
  });

  it('mounts a nested SchemaDrivenPlugin for a related route segment', () => {
    activeSchema = backupsSchema;
    renderBackupsPlugin('/apps/mysql_backups/restores');

    expect(screen.getByRole('tab', { name: 'Restore' })).toHaveAttribute(
      'aria-selected',
      'true'
    );
    expect(screen.getByText('list:mysql_backups/restore')).toBeInTheDocument();
    expect(screen.queryByText('list:mysql_backups')).toBeNull();
  });
});

describe('SchemaDrivenPlugin — write access', () => {
  it('renders the entity edit form for a session that may mutate', () => {
    renderEdit();

    expect(
      screen.queryByTestId('plugin-entity-edit-read-only')
    ).not.toBeInTheDocument();
    expect(screen.getByText(/Edit Nodes #5/)).toBeInTheDocument();
  });

  it('renders the read-only guard instead of the entity edit form for a non-admin', () => {
    mockCanMutate = false;
    renderEdit();

    expect(
      screen.getByTestId('plugin-entity-edit-read-only')
    ).toBeInTheDocument();
    expect(screen.queryByText(/Edit Nodes #5/)).not.toBeInTheDocument();
  });
});
