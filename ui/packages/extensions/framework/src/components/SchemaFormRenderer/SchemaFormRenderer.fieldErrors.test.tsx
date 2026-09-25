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
import { useFormContext, type UseFormReturn } from 'react-hook-form';
import type { FieldValidationError } from '@pmm-extensions/api';
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
// rendered field type surfaces its mapped error inline. The described fields
// pin `help_placement` so the help icon is what is under test here — left to
// the default, a description this short would render inline instead.
const SECTIONS: FormSection[] = [
  {
    title: 'Task',
    fields: [
      {
        type: 'string',
        name: 'title',
        label: 'Title',
        description: 'A title',
        help_placement: 'tooltip',
      },
      {
        type: 'integer',
        name: 'limit',
        label: 'Row Limit',
        description: 'Max rows',
        help_placement: 'tooltip',
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
  it.each([false, true])(
    'reactively reveals fail rules and highlights every named field (advanced=%s)',
    async (advanced) => {
      const user = userEvent.setup();
      const onSubmit = vi.fn();
      renderWithProviders(
        <SchemaFormRenderer
          sections={[
            {
              title: 'Main',
              fields: [{ type: 'bool', name: 'blocked', label: 'Blocked' }],
            },
            {
              title: 'Options',
              advanced,
              collapsible: true,
              collapsed_by_default: true,
              fail_when: [
                {
                  fail_when: { truthy: 'blocked' },
                  error_fields: ['options.first', 'second'],
                  message: 'Invalid combination.',
                },
              ],
              fields: [
                { type: 'string', name: 'options.first', label: 'First' },
                { type: 'string', name: 'second', label: 'Second' },
              ],
            },
          ]}
          onSubmit={onSubmit}
        />
      );

      expect(
        screen.queryByRole('textbox', { name: 'First' })
      ).not.toBeInTheDocument();
      await user.click(screen.getByLabelText('Blocked'));
      expect(screen.getByRole('button', { name: 'Options' })).toHaveAttribute(
        'aria-expanded',
        'true'
      );
      expect(
        screen.queryByTestId('show-advanced-options')
      ).not.toBeInTheDocument();
      expect(screen.getByRole('alert')).toHaveTextContent(
        'Invalid combination.'
      );
      for (const name of ['First', 'Second']) {
        expect(screen.getByRole('textbox', { name })).toHaveAttribute(
          'aria-invalid',
          'true'
        );
        expect(
          screen.getByRole('textbox', { name })
        ).toHaveAccessibleDescription('Invalid combination.');
      }
      await user.click(screen.getByRole('button', { name: 'Run' }));
      expect(onSubmit).not.toHaveBeenCalled();
      await user.type(
        screen.getByRole('textbox', { name: 'First' }),
        'still blocked'
      );
      expect(screen.getByRole('textbox', { name: 'First' })).toHaveAttribute(
        'aria-invalid',
        'true'
      );

      await user.click(screen.getByLabelText('Blocked'));
      expect(
        screen.queryByText('Invalid combination.')
      ).not.toBeInTheDocument();
      for (const name of ['First', 'Second']) {
        expect(screen.getByRole('textbox', { name })).toHaveAttribute(
          'aria-invalid',
          'false'
        );
      }
      await user.click(screen.getByRole('button', { name: 'Run' }));
      await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
    }
  );

  it('skips unknown and unmounted error fields without wedging submission', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    let form: UseFormReturn<Record<string, unknown>> | undefined;
    function Probe({ children }: { children: ReactNode }) {
      form = useFormContext();
      return children;
    }
    renderWithProviders(
      <SchemaFormRenderer
        sections={[
          {
            title: 'Main',
            fail_when: [
              {
                fail_when: { truthy: 'blocked' },
                error_fields: ['ghost', 'hidden', 'unmounted'],
                message: 'Blocked combination.',
              },
            ],
            fields: [
              { type: 'bool', name: 'blocked', label: 'Blocked' },
              {
                type: 'string',
                name: 'hidden',
                label: 'Hidden',
                forbidden: [{ when: { truthy: 'blocked' } }],
              },
            ],
          },
          {
            title: 'Unopened',
            collapsible: true,
            collapsed_by_default: true,
            fields: [{ type: 'string', name: 'unmounted', label: 'Unmounted' }],
          },
        ]}
        renderField={({ renderDefault }) => <Probe>{renderDefault()}</Probe>}
        onSubmit={onSubmit}
      />
    );

    await user.click(screen.getByLabelText('Blocked'));
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Blocked combination.'
    );
    expect(
      screen.queryByRole('textbox', { name: 'Hidden' })
    ).not.toBeInTheDocument();
    expect(form).toBeDefined();
    for (const path of ['ghost', 'hidden']) {
      expect(form?.getFieldState(path).error).toBeUndefined();
    }
    // `unmounted` lives in a different section than the rule that targets it,
    // and the section auto-expands to reveal it — no manual click needed.
    expect(screen.getByRole('button', { name: 'Unopened' })).toHaveAttribute(
      'aria-expanded',
      'true'
    );
    await waitFor(() =>
      expect(
        screen.getByRole('textbox', { name: 'Unmounted' })
      ).toHaveAccessibleDescription('Blocked combination.')
    );
    // Collapsing it back unmounts the field; its inline error clears with it.
    await user.click(screen.getByRole('button', { name: 'Unopened' }));
    await waitFor(() =>
      expect(form?.getFieldState('unmounted').error).toBeUndefined()
    );
    await user.click(screen.getByLabelText('Blocked'));
    await user.click(screen.getByRole('button', { name: 'Run' }));
    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
  });

  it.each([false, true])(
    "reveals a fail rule's target section across sections even when it starts collapsed (advanced=%s)",
    async (advanced) => {
      const user = userEvent.setup();
      renderWithProviders(
        <SchemaFormRenderer
          sections={[
            {
              title: 'Main',
              fields: [{ type: 'bool', name: 'blocked', label: 'Blocked' }],
              fail_when: [
                {
                  fail_when: { truthy: 'blocked' },
                  error_fields: ['target'],
                  message: 'Cross-section violation.',
                },
              ],
            },
            {
              title: 'Options',
              advanced,
              collapsible: true,
              collapsed_by_default: true,
              fields: [{ type: 'string', name: 'target', label: 'Target' }],
            },
          ]}
          onSubmit={() => {}}
        />
      );

      expect(
        screen.queryByRole('textbox', { name: 'Target' })
      ).not.toBeInTheDocument();
      await user.click(screen.getByLabelText('Blocked'));

      expect(
        screen.queryByTestId('show-advanced-options')
      ).not.toBeInTheDocument();
      expect(screen.getByRole('button', { name: 'Options' })).toHaveAttribute(
        'aria-expanded',
        'true'
      );
      const target = await screen.findByRole('textbox', { name: 'Target' });
      await waitFor(() =>
        expect(target).toHaveAttribute('aria-invalid', 'true')
      );
      expect(target).toHaveAccessibleDescription('Cross-section violation.');

      await user.click(screen.getByLabelText('Blocked'));
      await waitFor(() =>
        expect(target).toHaveAttribute('aria-invalid', 'false')
      );
    }
  );

  it('applies an active fail rule after native validation clears', async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <SchemaFormRenderer
        sections={[
          {
            title: 'Main',
            fields: [
              { type: 'bool', name: 'blocked', label: 'Blocked' },
              {
                type: 'string',
                name: 'target',
                label: 'Target',
                required: true,
              },
            ],
            fail_when: [
              {
                fail_when: { truthy: 'blocked' },
                error_fields: ['target'],
                message: 'Rule violation.',
              },
            ],
          },
        ]}
        onSubmit={() => {}}
      />
    );
    await user.click(screen.getByRole('button', { name: 'Run' }));
    const target = screen.getByRole('textbox', { name: /Target/ });
    expect(target).toHaveAccessibleDescription('Target is required');
    await user.click(screen.getByLabelText('Blocked'));
    expect(target).toHaveAccessibleDescription('Target is required');
    await user.type(target, 'valid');
    await waitFor(() =>
      expect(target).toHaveAccessibleDescription('Rule violation.')
    );
    await user.click(screen.getByLabelText('Blocked'));
    expect(target).toHaveAttribute('aria-invalid', 'false');
  });

  it('keeps an error until the last rule targeting the field clears, even after submit', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    renderWithProviders(
      <SchemaFormRenderer
        sections={[
          {
            title: 'Main',
            fields: [
              { type: 'bool', name: 'first', label: 'First rule' },
              { type: 'bool', name: 'second', label: 'Second rule' },
              { type: 'string', name: 'target', label: 'Target' },
            ],
            fail_when: [
              {
                fail_when: { truthy: 'first' },
                error_fields: ['target'],
                message: 'First violation.',
              },
              {
                fail_when: { truthy: 'second' },
                error_fields: ['target'],
                message: 'Second violation.',
              },
            ],
          },
        ]}
        onSubmit={onSubmit}
      />
    );

    await user.click(screen.getByRole('button', { name: 'Run' }));
    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
    await user.click(screen.getByLabelText('First rule'));
    await user.click(screen.getByLabelText('Second rule'));
    const target = screen.getByRole('textbox', { name: 'Target' });
    expect(target).toHaveAccessibleDescription('First violation.');
    await user.type(target, 'does not change either predicate');
    await waitFor(() =>
      expect(target).toHaveAccessibleDescription('First violation.')
    );
    await user.click(screen.getByLabelText('First rule'));
    expect(target).toHaveAccessibleDescription('Second violation.');
    await user.click(screen.getByLabelText('Second rule'));
    expect(target).toHaveAttribute('aria-invalid', 'false');
  });

  it('does not clear a server error that replaces a fail-rule error', async () => {
    const user = userEvent.setup();
    const sections: FormSection[] = [
      {
        title: 'Main',
        fields: [
          { type: 'bool', name: 'blocked', label: 'Blocked' },
          { type: 'string', name: 'target', label: 'Target' },
        ],
        fail_when: [
          {
            fail_when: { truthy: 'blocked' },
            error_fields: ['target'],
            message: 'Rule violation.',
          },
        ],
      },
    ];
    function Wrapper() {
      const [errors, setErrors] = useState<FieldValidationError[]>([]);
      return (
        <>
          <button
            type="button"
            onClick={() =>
              setErrors([{ path: 'target', message: 'Server rejection.' }])
            }
          >
            Server response
          </button>
          <SchemaFormRenderer
            sections={sections}
            onSubmit={() => {}}
            fieldErrors={errors}
          />
        </>
      );
    }
    renderWithProviders(<Wrapper />);
    await user.click(screen.getByLabelText('Blocked'));
    expect(
      screen.getByRole('textbox', { name: 'Target' })
    ).toHaveAccessibleDescription('Rule violation.');
    await user.click(screen.getByRole('button', { name: 'Server response' }));
    await user.click(screen.getByLabelText('Blocked'));
    expect(
      screen.getByRole('textbox', { name: 'Target' })
    ).toHaveAccessibleDescription('Server rejection.');
  });

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

  it('does not re-invoke the fail-error render override on an unrelated re-render', () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false, staleTime: Infinity } },
    });
    const onSubmit = () => {};
    const renderFieldSpy = vi.fn(({ renderDefault }) => renderDefault());
    const sections: FormSection[] = [
      {
        title: 'Main',
        fields: [
          { type: 'bool', name: 'blocked', label: 'Blocked' },
          { type: 'string', name: 'target', label: 'Target' },
        ],
        fail_when: [
          {
            fail_when: { truthy: 'blocked' },
            error_fields: ['target'],
            message: 'Violation.',
          },
        ],
      },
    ];
    const { rerender } = render(
      <QueryClientProvider client={queryClient}>
        <SchemaFormRenderer
          sections={sections}
          onSubmit={onSubmit}
          renderField={renderFieldSpy}
        />
      </QueryClientProvider>
    );
    const callsAfterMount = renderFieldSpy.mock.calls.length;

    rerender(
      <QueryClientProvider client={queryClient}>
        <SchemaFormRenderer
          sections={sections}
          onSubmit={onSubmit}
          renderField={renderFieldSpy}
          loading
        />
      </QueryClientProvider>
    );

    expect(renderFieldSpy.mock.calls.length).toBe(callsAfterMount);
  });
});
