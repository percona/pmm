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
import { describe, expect, it, vi } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import { SnackbarProvider } from 'notistack';
import type { PluginSchema } from '@sep/api';
import { PluginCreatePage } from './PluginCreatePage';
import type { RenderFormSlot } from './types';

const mockCreateTaskMutate = vi.fn();

vi.mock('@sep/api', () => ({
  useCreatePluginTask: () => ({
    mutate: mockCreateTaskMutate,
    isPending: false,
  }),
  useCreatePluginEntity: () => ({ mutate: vi.fn(), isPending: false }),
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
        { type: 'string', name: 'title', label: 'Title' },
        { type: 'integer', name: 'count', label: 'Count' },
      ],
    },
  ],
} as unknown as PluginSchema;

function renderPage(extra?: { renderCreateForm?: RenderFormSlot }) {
  return render(
    <SnackbarProvider>
      <MemoryRouter initialEntries={['/apps/checksums/new']}>
        <PluginCreatePage schema={schema} pluginName="checksums" {...extra} />
      </MemoryRouter>
    </SnackbarProvider>
  );
}

describe('PluginCreatePage — renderCreateForm slot', () => {
  it('renders the default SchemaFormRenderer when no slot is supplied', () => {
    renderPage();
    // Default form body: the schema field renders, with the framework chrome.
    expect(screen.getByText('New Checksum')).toBeInTheDocument();
    expect(screen.getByLabelText('Title')).toBeInTheDocument();
    expect(screen.queryByTestId('custom-form')).toBeNull();
  });

  it('replaces only the form body with the slot, keeping the page chrome', () => {
    const renderCreateForm: RenderFormSlot = () => (
      <div data-testid="custom-form">custom body</div>
    );
    renderPage({ renderCreateForm });
    // Chrome preserved (title); default form body replaced.
    expect(screen.getByText('New Checksum')).toBeInTheDocument();
    expect(screen.getByTestId('custom-form')).toBeInTheDocument();
    expect(screen.queryByLabelText('Title')).toBeNull();
  });

  it('falls back to the default form when the slot returns a nullish value', () => {
    const renderCreateForm: RenderFormSlot = () => null;
    renderPage({ renderCreateForm });
    // null slot → default SchemaFormRenderer renders the schema fields.
    expect(screen.getByLabelText('Title')).toBeInTheDocument();
    expect(screen.queryByTestId('custom-form')).toBeNull();
  });

  it('wires the framework submit handler into the slot and coerces the payload', async () => {
    const user = userEvent.setup();
    // A custom slot that bypasses SchemaFormRenderer submits raw string values;
    // the page submit boundary must still coerce `count` to a number.
    const renderCreateForm: RenderFormSlot = ({ onSubmit, loading }) => (
      <button
        type="button"
        disabled={loading}
        onClick={() => onSubmit({ title: 'from-slot', count: '7' })}
      >
        Submit slot
      </button>
    );
    renderPage({ renderCreateForm });

    await user.click(screen.getByRole('button', { name: 'Submit slot' }));
    await waitFor(() => expect(mockCreateTaskMutate).toHaveBeenCalledTimes(1));
    expect(mockCreateTaskMutate).toHaveBeenCalledWith(
      { title: 'from-slot', count: 7 },
      expect.objectContaining({
        onSuccess: expect.any(Function),
        onError: expect.any(Function),
      })
    );
  });
});
