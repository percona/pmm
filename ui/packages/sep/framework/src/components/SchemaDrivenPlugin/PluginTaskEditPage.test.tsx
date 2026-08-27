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
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { SnackbarProvider } from 'notistack';
import { ApiError, type PluginSchema } from '@sep/api';
import {
  PluginTaskEditPage,
  normalizeChoiceDefaults,
} from './PluginTaskEditPage';
import type { RenderFormSlot } from './types';

const mockUpdateTaskMutate = vi.fn();
const mockUsePluginTask = vi.fn();

/** Flipped per test to cover the read-only (non-admin) rendering. */
let mockCanMutate = true;

vi.mock('@sep/api', () => ({
  useAuth: () => ({ isAdmin: mockCanMutate, canMutate: mockCanMutate }),
  useUpdatePluginTask: () => ({
    mutate: mockUpdateTaskMutate,
    isPending: false,
    isError: false,
    error: null,
  }),
  usePluginTask: (...args: unknown[]) => mockUsePluginTask(...args),
  useAlertConfig: () => ({
    data: { available: true },
    isLoading: false,
    isError: false,
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

const schema: PluginSchema = {
  pluginName: 'checksums',
  display_name: 'Checksum',
  description: 'Test',
  capabilities: {},
  list_view: { columns: [{ key: 'name', label: 'Name' }] },
  forms: [
    {
      title: 'Main',
      fields: [
        { type: 'string', name: 'task_name', label: 'Task Name' },
        { type: 'string', name: 'title', label: 'Title' },
        { type: 'integer', name: 'count', label: 'Count' },
      ],
    },
  ],
} as unknown as PluginSchema;

const STORED_TASK = {
  id: 1,
  name: 'check1',
  status: 'completed',
  data: { _form: { task_name: 'check1', title: 'hello', count: 7 } },
};

function renderAt(
  extra?: { renderEditForm?: RenderFormSlot },
  path = '/apps/checksums/task/check1/edit'
) {
  return render(
    <SnackbarProvider>
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route
            path="/apps/:plugin/task/:id/edit"
            element={
              <PluginTaskEditPage
                schema={schema}
                pluginName="checksums"
                {...extra}
              />
            }
          />
          <Route
            path="/apps/:plugin/task/:id"
            element={<div>detail page</div>}
          />
        </Routes>
      </MemoryRouter>
    </SnackbarProvider>
  );
}

beforeEach(() => {
  mockUpdateTaskMutate.mockReset();
  mockUsePluginTask.mockReset();
  mockCanMutate = true;
});

describe('PluginTaskEditPage', () => {
  it('prefills the editable fields from the stored create-form body', () => {
    mockUsePluginTask.mockReturnValue({ data: STORED_TASK, isLoading: false });

    renderAt();

    expect(screen.getByText('Edit Checksum: check1')).toBeInTheDocument();
    expect(screen.getByLabelText('Title')).toHaveValue('hello');
  });

  it('renders the task name read-only (no editable task_name input)', () => {
    mockUsePluginTask.mockReturnValue({ data: STORED_TASK, isLoading: false });

    renderAt();

    // The immutable identity is shown in the header, never as an editable field.
    expect(screen.queryByLabelText('Task Name')).toBeNull();
    expect(screen.getByText('Edit Checksum: check1')).toBeInTheDocument();
  });

  it('submits coerced values and pins task_name to the route id', async () => {
    mockUsePluginTask.mockReturnValue({ data: STORED_TASK, isLoading: false });
    // A slot bypassing SchemaFormRenderer submits raw string values and omits
    // task_name; the page coerces `count` and re-asserts the original name.
    const renderEditForm: RenderFormSlot = ({ onSubmit, loading }) => (
      <button
        type="button"
        disabled={loading}
        onClick={() => onSubmit({ title: 'changed', count: '9' })}
      >
        Submit slot
      </button>
    );

    renderAt({ renderEditForm });
    await userEvent.click(screen.getByRole('button', { name: 'Submit slot' }));

    await waitFor(() => expect(mockUpdateTaskMutate).toHaveBeenCalledTimes(1));
    expect(mockUpdateTaskMutate).toHaveBeenCalledWith(
      {
        taskId: 'check1',
        values: { title: 'changed', count: 9, task_name: 'check1' },
      },
      expect.objectContaining({
        onSuccess: expect.any(Function),
        onError: expect.any(Function),
      })
    );
  });

  it('keeps task_name pinned even when the submitted body carries a different name', async () => {
    mockUsePluginTask.mockReturnValue({ data: STORED_TASK, isLoading: false });
    const renderEditForm: RenderFormSlot = ({ onSubmit }) => (
      <button
        type="button"
        onClick={() => onSubmit({ task_name: 'renamed', title: 'x' })}
      >
        Submit slot
      </button>
    );

    renderAt({ renderEditForm });
    await userEvent.click(screen.getByRole('button', { name: 'Submit slot' }));

    await waitFor(() => expect(mockUpdateTaskMutate).toHaveBeenCalledTimes(1));
    const [{ values }] = mockUpdateTaskMutate.mock.calls[0];
    expect(values.task_name).toBe('check1');
  });

  it('navigates back to the task detail on a successful update', async () => {
    mockUsePluginTask.mockReturnValue({ data: STORED_TASK, isLoading: false });
    mockUpdateTaskMutate.mockImplementation((_vars, opts) =>
      opts.onSuccess?.()
    );
    const renderEditForm: RenderFormSlot = ({ onSubmit }) => (
      <button type="button" onClick={() => onSubmit({ title: 'x' })}>
        Submit slot
      </button>
    );

    renderAt({ renderEditForm });
    await userEvent.click(screen.getByRole('button', { name: 'Submit slot' }));

    await waitFor(() =>
      expect(screen.getByText('detail page')).toBeInTheDocument()
    );
  });

  it('redirects to detail without crashing when the task has no stored form', () => {
    mockUsePluginTask.mockReturnValue({
      data: { id: 1, name: 'check1', status: 'completed', data: { meta: {} } },
      isLoading: false,
    });

    renderAt();

    expect(screen.getByText('detail page')).toBeInTheDocument();
    expect(screen.queryByText('Edit Checksum: check1')).toBeNull();
  });

  it('threads capabilities so a stored alert_on_fail survives an edit', async () => {
    // `alert_on_fail` is excluded from the schema sections and only rendered
    // (and seeded) when the capability is passed through. Without it the field
    // is dropped from the submission and the backend resets it to its default.
    const alertSchema = {
      ...schema,
      capabilities: { alert_on_fail: true },
    } as unknown as PluginSchema;
    mockUsePluginTask.mockReturnValue({
      data: {
        id: 1,
        name: 'check1',
        status: 'completed',
        data: {
          _form: {
            task_name: 'check1',
            title: 'hello',
            count: 7,
            alert_on_fail: true,
          },
        },
      },
      isLoading: false,
    });

    render(
      <SnackbarProvider>
        <MemoryRouter initialEntries={['/apps/checksums/task/check1/edit']}>
          <Routes>
            <Route
              path="/apps/:plugin/task/:id/edit"
              element={
                <PluginTaskEditPage
                  schema={alertSchema}
                  pluginName="checksums"
                />
              }
            />
            <Route
              path="/apps/:plugin/task/:id"
              element={<div>detail page</div>}
            />
          </Routes>
        </MemoryRouter>
      </SnackbarProvider>
    );

    await userEvent.click(screen.getByRole('button', { name: 'Save' }));

    await waitFor(() => expect(mockUpdateTaskMutate).toHaveBeenCalledTimes(1));
    const [{ values }] = mockUpdateTaskMutate.mock.calls[0];
    expect(values.alert_on_fail).toBe(true);
  });

  it('normalizes lowercase multi_choice stored values to match uppercase schema choices', async () => {
    // Reproduces the mysql_backups 422: model_dump(mode="json") on a StrEnum
    // with auto() stores lowercase values ("rsync") while schema choices use the
    // member name ("RSYNC"). The mismatch left the multi-select empty on load,
    // causing the form to submit upload:[] and the backend to return 422.
    const uploadSchema: PluginSchema = {
      pluginName: 'mysql_backups',
      display_name: 'MySQL Backup',
      description: 'Test',
      capabilities: {},
      list_view: { columns: [{ key: 'name', label: 'Name' }] },
      forms: [
        {
          title: 'Main',
          fields: [
            { type: 'string', name: 'task_name', label: 'Task Name' },
            {
              type: 'multi_choice',
              name: 'upload',
              label: 'Upload',
              required: true,
              min_items: 1,
              choices: [
                { value: 'RSYNC', label: 'Rsync' },
                { value: 'S3', label: 'S3' },
              ],
            },
          ],
        },
      ],
    } as unknown as PluginSchema;

    mockUsePluginTask.mockReturnValue({
      data: {
        id: 1,
        name: 'backup1',
        status: 'completed',
        // Stored with lowercase "rsync" (Python StrEnum auto() output).
        data: { _form: { task_name: 'backup1', upload: ['rsync'] } },
      },
      isLoading: false,
    });
    mockUpdateTaskMutate.mockImplementation((_vars, opts) =>
      opts.onSuccess?.()
    );

    render(
      <SnackbarProvider>
        <MemoryRouter
          initialEntries={['/apps/mysql_backups/task/backup1/edit']}
        >
          <Routes>
            <Route
              path="/apps/:plugin/task/:id/edit"
              element={
                <PluginTaskEditPage
                  schema={uploadSchema}
                  pluginName="mysql_backups"
                />
              }
            />
            <Route
              path="/apps/:plugin/task/:id"
              element={<div>detail page</div>}
            />
          </Routes>
        </MemoryRouter>
      </SnackbarProvider>
    );

    await userEvent.click(screen.getByRole('button', { name: 'Save' }));

    await waitFor(() => expect(mockUpdateTaskMutate).toHaveBeenCalledTimes(1));
    const [{ values }] = mockUpdateTaskMutate.mock.calls[0];
    // After normalization, the submitted value must use the canonical "RSYNC",
    // not the stored "rsync", so the backend accepts the request.
    expect(values.upload).toEqual(['RSYNC']);
  });
});

describe('normalizeChoiceDefaults', () => {
  const sections = [
    {
      title: 'Main',
      fields: [
        {
          type: 'multi_choice' as const,
          name: 'upload',
          label: 'Upload',
          required: true,
          choices: [
            { value: 'RSYNC', label: 'Rsync' },
            { value: 'S3', label: 'S3' },
          ],
        },
        {
          type: 'choice' as const,
          name: 'mode',
          label: 'Mode',
          required: false,
          choices: [
            { value: 'FULL', label: 'Full' },
            { value: 'INCREMENTAL', label: 'Incremental' },
          ],
        },
        {
          type: 'string' as const,
          name: 'title',
          label: 'Title',
          required: false,
        },
      ],
    },
  ];

  it('upcases lowercase multi_choice values to canonical schema values', () => {
    const result = normalizeChoiceDefaults(
      { upload: ['rsync', 's3'] },
      sections
    );
    expect(result.upload).toEqual(['RSYNC', 'S3']);
  });

  it('normalizes a single choice field value', () => {
    const result = normalizeChoiceDefaults({ mode: 'full' }, sections);
    expect(result.mode).toBe('FULL');
  });

  it('leaves non-choice fields untouched', () => {
    const result = normalizeChoiceDefaults({ title: 'hello' }, sections);
    expect(result.title).toBe('hello');
  });

  it('preserves unrecognised values so validation can surface them', () => {
    const result = normalizeChoiceDefaults({ upload: ['unknown'] }, sections);
    expect(result.upload).toEqual(['unknown']);
  });

  it('handles already-canonical values without modification', () => {
    const result = normalizeChoiceDefaults({ upload: ['RSYNC'] }, sections);
    expect(result.upload).toEqual(['RSYNC']);
  });

  it('normalizes a one-of branch field stored at its dotted path', () => {
    const oneOfSections = [
      {
        title: 'Source',
        fields: [
          {
            type: 'one_of' as const,
            name: 'source',
            label: 'Source',
            discriminator: 'source.mode',
            branches: [
              {
                value: 'rsync',
                label: 'Rsync',
                fields: [
                  {
                    type: 'choice' as const,
                    name: 'source.transport',
                    label: 'Transport',
                    required: false,
                    choices: [
                      { value: 'SSH', label: 'SSH' },
                      { value: 'DAEMON', label: 'Daemon' },
                    ],
                  },
                ],
              },
            ],
          },
        ],
      },
    ];
    const form = { source: { mode: 'rsync', transport: 'ssh' } };

    const result = normalizeChoiceDefaults(form, oneOfSections);

    expect(result.source).toEqual({ mode: 'rsync', transport: 'SSH' });
    // The input is not mutated: `setAtPath` clones the intermediates it walks.
    expect(form.source.transport).toBe('ssh');
  });
});

describe('PluginTaskEditPage — write access', () => {
  it('renders the edit form for a session that may mutate', () => {
    mockUsePluginTask.mockReturnValue({
      data: {
        name: 'check1',
        data: { _form: { task_name: 'check1', title: 'Nightly' } },
      },
      isLoading: false,
    });

    renderAt();

    expect(screen.getByRole('button', { name: 'Save' })).toBeInTheDocument();
    expect(
      screen.queryByTestId('plugin-task-edit-read-only')
    ).not.toBeInTheDocument();
  });

  it('renders the read-only guard instead of the edit form for a non-admin', () => {
    mockCanMutate = false;
    mockUsePluginTask.mockReturnValue({
      data: {
        name: 'check1',
        data: { _form: { task_name: 'check1', title: 'Nightly' } },
      },
      isLoading: false,
    });

    renderAt();

    expect(
      screen.getByTestId('plugin-task-edit-read-only')
    ).toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: 'Save' })
    ).not.toBeInTheDocument();
  });
});
