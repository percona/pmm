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

import { describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useState, type ReactNode } from 'react';
import type { FieldValidationError } from '@sep/api';
import { SchemaFormRenderer } from './SchemaFormRenderer';
import type { FormSection } from './types';

function renderWithProviders(ui: ReactNode) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Infinity } },
  });
  return render(
    <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>
  );
}

// One field per helperText-based component plus a choice field, to prove every
// rendered field type surfaces its mapped error inline.
const SECTIONS: FormSection[] = [
  {
    title: 'Task',
    fields: [
      { type: 'string', name: 'title', label: 'Title', description: 'A title' },
      {
        type: 'integer',
        name: 'limit',
        label: 'Row Limit',
        description: 'Max rows',
      },
      {
        type: 'choice',
        name: 'mode',
        label: 'Mode',
        choices: [
          { label: 'A', value: 'a' },
          { label: 'B', value: 'b' },
        ],
      },
    ],
  },
];

describe('SchemaFormRenderer field errors', () => {
  it('applies fieldErrors inline on helperText-based fields and shows the banner', async () => {
    renderWithProviders(
      <SchemaFormRenderer
        sections={SECTIONS}
        onSubmit={() => {}}
        submitError={'Failed\n• Title: must not be blank'}
        fieldErrors={[
          { path: 'title', message: 'must not be blank' },
          { path: 'limit', message: 'must be positive' },
        ]}
      />
    );

    await waitFor(() => {
      expect(screen.getByText('must not be blank')).toBeInTheDocument();
    });
    // Inline error replaces the field description helperText on the string field.
    expect(screen.queryByText('A title')).not.toBeInTheDocument();
    // Integer (helperText) field also surfaces its mapped error.
    expect(screen.getByText('must be positive')).toBeInTheDocument();
    expect(screen.queryByText('Max rows')).not.toBeInTheDocument();
    // Help icons remain while helperText shows the error; pin count + visible <label>.
    expect(document.querySelectorAll('[data-help-for="Title"]')).toHaveLength(
      2
    );
    expect(
      document.querySelectorAll('label [data-help-for="Title"]')
    ).toHaveLength(1);
    expect(
      document.querySelectorAll('[data-help-for="Row Limit"]')
    ).toHaveLength(2);
    expect(
      document.querySelectorAll('label [data-help-for="Row Limit"]')
    ).toHaveLength(1);
    // The persistent banner is rendered.
    expect(screen.getByText(/Failed/)).toBeInTheDocument();
  });

  it('clears a stale server error when a later submit no longer reports it', async () => {
    function Wrapper() {
      const [errors, setErrors] = useState<FieldValidationError[]>([
        { path: 'title', message: 'title bad' },
        { path: 'limit', message: 'limit bad' },
      ]);
      return (
        <>
          <button
            type="button"
            onClick={() => setErrors([{ path: 'limit', message: 'limit bad' }])}
          >
            resubmit
          </button>
          <SchemaFormRenderer
            sections={SECTIONS}
            onSubmit={() => {}}
            fieldErrors={errors}
          />
        </>
      );
    }
    renderWithProviders(<Wrapper />);

    await waitFor(() =>
      expect(screen.getByText('title bad')).toBeInTheDocument()
    );

    screen.getByText('resubmit').click();

    await waitFor(() =>
      expect(screen.queryByText('title bad')).not.toBeInTheDocument()
    );
    expect(screen.getByText('limit bad')).toBeInTheDocument();
  });

  it('keeps a rendered-field server error inline when a resubmit is blocked client-side', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    // `title` is required, so emptying it blocks the resubmit before onSubmit runs.
    const sections: FormSection[] = [
      {
        title: 'Task',
        fields: [
          {
            type: 'string',
            name: 'title',
            label: 'Title',
            description: 'A title',
            required: true,
          },
          {
            type: 'integer',
            name: 'limit',
            label: 'Row Limit',
            description: 'Max rows',
          },
        ],
      },
    ];
    renderWithProviders(
      <SchemaFormRenderer
        sections={sections}
        onSubmit={onSubmit}
        submitLabel="Run"
        fieldErrors={[{ path: 'limit', message: 'must be positive' }]}
      />
    );

    await waitFor(() =>
      expect(screen.getByText('must be positive')).toBeInTheDocument()
    );

    // Empty the required title field, then resubmit: the client-side gate blocks
    // the submit, so the eager clear must not drop `limit`'s inline highlight.
    // Prefer role+name: getByLabelText(/Title/) also matches "Help for Title".
    await user.clear(screen.getByRole('textbox', { name: /Title/ }));
    await user.click(screen.getByRole('button', { name: 'Run' }));

    await waitFor(() =>
      expect(screen.getByText('Title is required')).toBeInTheDocument()
    );
    expect(onSubmit).not.toHaveBeenCalled();
    // `limit`'s server error survives the blocked resubmit, in sync with the banner.
    expect(screen.getByText('must be positive')).toBeInTheDocument();
  });

  it('does not wedge resubmission when a fieldError lands on a non-rendered field', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    renderWithProviders(
      <SchemaFormRenderer
        sections={SECTIONS}
        onSubmit={onSubmit}
        submitLabel="Run"
        // No 'ghost' field is rendered, so setError on it leaves an orphan in
        // formState.errors that handleSubmit would otherwise refuse to step past.
        fieldErrors={[{ path: 'ghost', message: 'rejected by server' }]}
      />
    );

    await user.click(screen.getByRole('button', { name: 'Run' }));

    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
  });
});
