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
import type { ReactNode } from 'react';
import { SchemaFormRenderer } from './SchemaFormRenderer';
import type { FailRule, FormSection } from './types';

function renderWithProviders(ui: ReactNode) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Infinity } },
  });
  return render(
    <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>
  );
}

const GPG_MESSAGE = 'A GPG format needs encrypt or post_run_encrypt.';

// Shaped after MySQL Backups: the rule spans two sections, so it can only be
// declared at the schema root.
const SECTIONS: FormSection[] = [
  {
    title: 'General',
    fields: [{ type: 'string', name: 'title', label: 'Title' }],
  },
  {
    title: 'Encryption',
    fields: [
      { type: 'bool', name: 'gpg', label: 'GPG format' },
      { type: 'bool', name: 'encrypt', label: 'Encrypt backup' },
      { type: 'bool', name: 'post_run_encrypt', label: 'Encrypt after' },
    ],
  },
];

const GPG_RULE: FailRule = {
  fail_when: {
    all: [{ truthy: 'gpg' }, { all_falsy: ['encrypt', 'post_run_encrypt'] }],
  },
  error_fields: ['encrypt', 'post_run_encrypt'],
  message: GPG_MESSAGE,
};

describe('SchemaFormRenderer — root fail_when rules', () => {
  it('shows a violated root rule inline on each of its error fields', async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <SchemaFormRenderer
        sections={SECTIONS}
        failRules={[GPG_RULE]}
        onSubmit={() => {}}
      />
    );

    expect(screen.queryByText(GPG_MESSAGE)).toBeNull();

    await user.click(screen.getByLabelText('GPG format'));

    for (const label of ['Encrypt backup', 'Encrypt after']) {
      await waitFor(() =>
        expect(screen.getByLabelText(label)).toHaveAttribute(
          'aria-invalid',
          'true'
        )
      );
      expect(screen.getByLabelText(label)).toHaveAccessibleDescription(
        GPG_MESSAGE
      );
    }
    // Inline only: a rendered target needs no form-level banner too.
    expect(screen.queryByTestId('form-rule-violation')).toBeNull();
  });

  it('clears the inline message once the values satisfy the rule', async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <SchemaFormRenderer
        sections={SECTIONS}
        failRules={[GPG_RULE]}
        onSubmit={() => {}}
      />
    );

    await user.click(screen.getByLabelText('GPG format'));
    await waitFor(() =>
      expect(screen.getByLabelText('Encrypt backup')).toHaveAttribute(
        'aria-invalid',
        'true'
      )
    );

    await user.click(screen.getByLabelText('Encrypt backup'));

    await waitFor(() =>
      expect(screen.queryByText(GPG_MESSAGE)).not.toBeInTheDocument()
    );
    for (const label of ['Encrypt backup', 'Encrypt after']) {
      expect(screen.getByLabelText(label)).toHaveAttribute(
        'aria-invalid',
        'false'
      );
    }
  });

  it('blocks submit while the rule is violated and sends once it is fixed', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    renderWithProviders(
      <SchemaFormRenderer
        sections={SECTIONS}
        failRules={[GPG_RULE]}
        onSubmit={onSubmit}
      />
    );

    await user.click(screen.getByLabelText('GPG format'));
    await user.click(screen.getByRole('button', { name: 'Run' }));
    expect(onSubmit).not.toHaveBeenCalled();

    await user.click(screen.getByLabelText('Encrypt after'));
    await user.click(screen.getByRole('button', { name: 'Run' }));
    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
    expect(onSubmit.mock.calls[0][0]).toMatchObject({
      gpg: true,
      post_run_encrypt: true,
    });
  });

  it.each([
    ['no error fields', [] as string[]],
    ['only unknown error fields', ['not_a_field']],
  ])(
    'falls back to the form-level area when the rule names %s',
    async (_, errorFields) => {
      const user = userEvent.setup();
      const onSubmit = vi.fn();
      renderWithProviders(
        <SchemaFormRenderer
          sections={SECTIONS}
          failRules={[{ ...GPG_RULE, error_fields: errorFields }]}
          onSubmit={onSubmit}
        />
      );

      await user.click(screen.getByLabelText('GPG format'));

      expect(screen.getByTestId('form-rule-violation')).toHaveTextContent(
        GPG_MESSAGE
      );
      await user.click(screen.getByRole('button', { name: 'Run' }));
      expect(onSubmit).not.toHaveBeenCalled();

      await user.click(screen.getByLabelText('Encrypt backup'));
      await waitFor(() =>
        expect(screen.queryByTestId('form-rule-violation')).toBeNull()
      );
    }
  );

  it('falls back to the form-level area when every error field is gated out', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    renderWithProviders(
      <SchemaFormRenderer
        sections={[
          ...SECTIONS,
          {
            title: 'Key',
            forbidden: [{ when: { truthy: 'gpg' } }],
            fields: [{ type: 'string', name: 'key_file', label: 'Key file' }],
          },
        ]}
        failRules={[{ ...GPG_RULE, error_fields: ['key_file'] }]}
        onSubmit={onSubmit}
      />
    );

    await user.click(screen.getByLabelText('GPG format'));

    expect(screen.queryByLabelText('Key file')).toBeNull();
    expect(screen.getByTestId('form-rule-violation')).toHaveTextContent(
      GPG_MESSAGE
    );
    await user.click(screen.getByRole('button', { name: 'Run' }));
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it('does not fire on a value in a hidden section, which is not submitted', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    renderWithProviders(
      <SchemaFormRenderer
        sections={[
          {
            title: 'General',
            fields: [{ type: 'bool', name: 'logical', label: 'Logical' }],
          },
          {
            title: 'Mydumper',
            forbidden: [{ when: { falsy: 'logical' } }],
            fields: [
              {
                type: 'bool',
                name: 'mydumper_flag',
                label: 'Mydumper flag',
                default: true,
              },
            ],
          },
        ]}
        failRules={[
          {
            fail_when: {
              all: [{ falsy: 'logical' }, { truthy: 'mydumper_flag' }],
            },
            error_fields: ['mydumper_flag'],
            message: 'Mydumper options need a logical backup.',
          },
        ]}
        onSubmit={onSubmit}
      />
    );

    expect(screen.queryByLabelText('Mydumper flag')).toBeNull();
    expect(
      screen.queryByText('Mydumper options need a logical backup.')
    ).toBeNull();

    await user.click(screen.getByRole('button', { name: 'Run' }));
    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
    expect(onSubmit.mock.calls[0][0]).not.toHaveProperty('mydumper_flag');
  });

  it('keeps a section holding a root rule target open while it is violated', async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <SchemaFormRenderer
        sections={[
          SECTIONS[0],
          { ...SECTIONS[1], collapsible: true, collapsed_by_default: true },
        ]}
        failRules={[{ ...GPG_RULE, fail_when: { truthy: 'title' } }]}
        onSubmit={() => {}}
      />
    );

    expect(screen.queryByLabelText('Encrypt backup')).toBeNull();
    await user.type(screen.getByLabelText('Title'), 'x');

    const encrypt = await screen.findByLabelText('Encrypt backup');
    await waitFor(() =>
      expect(encrypt).toHaveAccessibleDescription(GPG_MESSAGE)
    );

    await user.click(screen.getByRole('button', { name: 'Encryption' }));
    expect(screen.getByLabelText('Encrypt backup')).toHaveAccessibleDescription(
      GPG_MESSAGE
    );
  });

  it('judges an inactive one-of branch in an unopened section as not submitted', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    renderWithProviders(
      <SchemaFormRenderer
        sections={[
          {
            title: 'Source',
            collapsible: true,
            collapsed_by_default: true,
            fields: [
              {
                type: 'one_of',
                name: 'source',
                label: 'Source',
                discriminator: 'source.mode',
                default: 'a',
                branches: [
                  {
                    value: 'a',
                    label: 'A',
                    fields: [{ type: 'string', name: 'source.a', label: 'A' }],
                  },
                  {
                    value: 'b',
                    label: 'B',
                    fields: [
                      {
                        type: 'bool',
                        name: 'source.b',
                        label: 'B flag',
                        default: true,
                      },
                    ],
                  },
                ],
              },
            ],
          },
        ]}
        failRules={[
          {
            fail_when: { truthy: 'source.b' },
            error_fields: [],
            message: 'B must stay off.',
          },
        ]}
        onSubmit={onSubmit}
      />
    );

    expect(screen.queryByTestId('form-rule-violation')).toBeNull();
    await user.click(screen.getByRole('button', { name: 'Run' }));
    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
    expect(onSubmit.mock.calls[0][0]).not.toHaveProperty(['source', 'b']);
  });

  it('leaves a schema without root rules to its section rules', async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    renderWithProviders(
      <SchemaFormRenderer
        sections={[
          {
            ...SECTIONS[1],
            fail_when: [GPG_RULE],
          },
        ]}
        onSubmit={onSubmit}
      />
    );

    await user.click(screen.getByLabelText('GPG format'));

    expect(screen.getByRole('alert')).toHaveTextContent(GPG_MESSAGE);
    expect(screen.queryByTestId('form-rule-violation')).toBeNull();
    await user.click(screen.getByRole('button', { name: 'Run' }));
    expect(onSubmit).not.toHaveBeenCalled();
  });
});
