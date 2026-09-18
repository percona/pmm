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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { FormProvider, useForm } from 'react-hook-form';
import type { PropsWithChildren } from 'react';
import { RemoteChoiceSelector } from './RemoteChoiceSelector';

vi.mock('@sep/api', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@sep/api')>()),
  apiClient: { get: vi.fn(), post: vi.fn() },
}));
import { apiClient } from '@sep/api';
const mocked = apiClient as unknown as { get: ReturnType<typeof vi.fn> };

const OPTIONS = [
  { value: 'backup-1', label: 'Backup 1' },
  {
    value: 'backup-2',
    label: 'Backup 2',
    disabled: true,
    disabled_reason: 'In progress',
  },
];

function makeClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false, retryDelay: 0, gcTime: 0, staleTime: 0 },
    },
  });
}

function Wrapper({
  children,
  client,
}: PropsWithChildren<{ client: QueryClient }>) {
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

interface HarnessProps {
  allowCustom?: boolean;
  dependsOn?: string;
  initialParent?: string | null;
  initialBackup?: string | null;
  required?: boolean;
}

function Harness({
  allowCustom,
  dependsOn,
  initialParent = null,
  initialBackup = null,
  required,
}: HarnessProps) {
  const methods = useForm<{ cluster: string | null; backup: string | null }>({
    defaultValues: { cluster: initialParent, backup: initialBackup },
  });
  const backup = methods.watch('backup');
  return (
    <FormProvider {...methods}>
      <RemoteChoiceSelector
        name="backup"
        label="Backup"
        required={required}
        endpointUrl="/apps/restore/backups"
        dependsOn={dependsOn}
        allowCustom={allowCustom}
      />
      <div data-testid="value">{JSON.stringify(backup ?? null)}</div>
      <div data-testid="error">
        {methods.formState.errors.backup?.message ?? ''}
      </div>
      <button
        type="button"
        onClick={() => methods.setValue('cluster', 'cluster-b')}
      >
        change-parent
      </button>
      <button
        type="button"
        onClick={() => methods.setValue('cluster', 'cluster-a')}
      >
        set-parent
      </button>
      <button type="button" onClick={() => void methods.trigger('backup')}>
        validate
      </button>
    </FormProvider>
  );
}

beforeEach(() => {
  mocked.get.mockReset();
  mocked.get.mockResolvedValue({ data: OPTIONS });
});

describe('RemoteChoiceSelector', () => {
  it('fetches and renders the Choice options', async () => {
    const user = userEvent.setup();
    render(
      <Wrapper client={makeClient()}>
        <Harness />
      </Wrapper>
    );
    await waitFor(() =>
      expect(mocked.get).toHaveBeenCalledWith('/apps/restore/backups')
    );

    await user.click(screen.getByRole('combobox'));
    expect(await screen.findByText('Backup 1')).toBeInTheDocument();
    expect(screen.getByText('Backup 2')).toBeInTheDocument();
  });

  it('commits the selected option value (a string) to the form', async () => {
    const user = userEvent.setup();
    render(
      <Wrapper client={makeClient()}>
        <Harness />
      </Wrapper>
    );
    await waitFor(() => expect(mocked.get).toHaveBeenCalled());

    await user.click(screen.getByRole('combobox'));
    await user.click(await screen.findByText('Backup 1'));

    expect(screen.getByTestId('value')).toHaveTextContent('"backup-1"');
  });

  it('renders a disabled option non-selectable with its reason tooltip', async () => {
    const user = userEvent.setup();
    render(
      <Wrapper client={makeClient()}>
        <Harness />
      </Wrapper>
    );
    await waitFor(() => expect(mocked.get).toHaveBeenCalled());

    await user.click(screen.getByRole('combobox'));
    const disabledLabel = await screen.findByText('Backup 2');
    expect(disabledLabel.closest('[role="option"]')).toHaveAttribute(
      'aria-disabled',
      'true'
    );

    await user.hover(disabledLabel);
    await waitFor(() => {
      expect(screen.getByRole('tooltip')).toHaveTextContent('In progress');
    });
  });

  it('renders empty-state helper text when the endpoint returns no options', async () => {
    mocked.get.mockResolvedValue({ data: [] });
    render(
      <Wrapper client={makeClient()}>
        <Harness />
      </Wrapper>
    );
    await waitFor(() =>
      expect(screen.getByText('No options available')).toBeInTheDocument()
    );
  });

  it('renders defensively when the endpoint returns a malformed (non-Choice) item', async () => {
    mocked.get.mockResolvedValue({
      data: [{ foo: 1 } as unknown as (typeof OPTIONS)[number]],
    });
    const user = userEvent.setup();
    render(
      <Wrapper client={makeClient()}>
        <Harness />
      </Wrapper>
    );
    await waitFor(() => expect(mocked.get).toHaveBeenCalled());
    // Opening the dropdown must not crash on an item missing value/label.
    await user.click(screen.getByRole('combobox'));
    expect(screen.getByRole('combobox')).toBeInTheDocument();
  });

  it('collapses a stored value with no matching option to empty in select mode', async () => {
    render(
      <Wrapper client={makeClient()}>
        <Harness initialBackup="ghost-value" />
      </Wrapper>
    );
    await waitFor(() => expect(mocked.get).toHaveBeenCalled());
    // Non-freeSolo MUI would warn on a bare string value; the input resolves to
    // empty rather than echoing the unmatched string.
    expect(screen.getByRole('combobox')).toHaveValue('');
  });

  it('surfaces an error-state helper text when the fetch fails', async () => {
    mocked.get.mockRejectedValue(new Error('boom'));
    render(
      <Wrapper client={makeClient()}>
        <Harness />
      </Wrapper>
    );
    await waitFor(() => expect(screen.getByText('boom')).toBeInTheDocument());
  });

  describe('cascade', () => {
    it('is disabled and skips the fetch until the parent has a value', async () => {
      render(
        <Wrapper client={makeClient()}>
          <Harness dependsOn="cluster" initialParent={null} />
        </Wrapper>
      );
      expect(screen.getByLabelText('Backup')).toBeDisabled();
      expect(screen.getByText('Select a value first')).toBeInTheDocument();
      expect(mocked.get).not.toHaveBeenCalled();
    });

    it('fetches with the parent value as a query param once the parent is set', async () => {
      render(
        <Wrapper client={makeClient()}>
          <Harness dependsOn="cluster" initialParent="cluster-a" />
        </Wrapper>
      );
      await waitFor(() =>
        expect(mocked.get).toHaveBeenCalledWith(
          '/apps/restore/backups?cluster=cluster-a'
        )
      );
      expect(screen.getByLabelText('Backup')).not.toBeDisabled();
    });

    it('treats a plain string-valued parent as present (not missing)', async () => {
      render(
        <Wrapper client={makeClient()}>
          <Harness dependsOn="cluster" initialParent="cluster-a" />
        </Wrapper>
      );
      // A string parent must not read as `isMissing`: the field enables and fetches.
      await waitFor(() =>
        expect(mocked.get).toHaveBeenCalledWith(
          '/apps/restore/backups?cluster=cluster-a'
        )
      );
      expect(
        screen.queryByText('Select a value first')
      ).not.toBeInTheDocument();
    });

    it('resets the child value when the parent value changes', async () => {
      const user = userEvent.setup();
      render(
        <Wrapper client={makeClient()}>
          <Harness
            dependsOn="cluster"
            initialParent="cluster-a"
            initialBackup="backup-1"
          />
        </Wrapper>
      );
      await waitFor(() => expect(mocked.get).toHaveBeenCalled());
      expect(screen.getByTestId('value')).toHaveTextContent('"backup-1"');

      await user.click(screen.getByRole('button', { name: 'change-parent' }));
      await waitFor(() =>
        expect(screen.getByTestId('value')).toHaveTextContent('null')
      );
    });

    it('keeps a free-typed value when the cascade parent is set afterwards', async () => {
      // Selecting the parent used to wipe backup_source, so Create submitted ""
      // and the backend answered 422 NonEmptyStr even though the input still
      // looked filled.
      const user = userEvent.setup();
      render(
        <Wrapper client={makeClient()}>
          <Harness dependsOn="cluster" initialParent={null} allowCustom />
        </Wrapper>
      );
      await user.type(screen.getByLabelText('Backup'), '/');
      await waitFor(() =>
        expect(screen.getByTestId('value')).toHaveTextContent('"/"')
      );

      await user.click(screen.getByRole('button', { name: 'set-parent' }));
      await waitFor(() =>
        expect(mocked.get).toHaveBeenCalledWith(
          '/apps/restore/backups?cluster=cluster-a'
        )
      );
      expect(screen.getByTestId('value')).toHaveTextContent('"/"');
    });

    it('stays typable with free-text entry while the parent has no value', async () => {
      // Options need the parent, but a free-typed value never does: disabling the
      // input would make a required allow_custom field unfillable (and push the
      // keystrokes into whichever field still holds focus).
      const user = userEvent.setup();
      render(
        <Wrapper client={makeClient()}>
          <Harness dependsOn="cluster" initialParent={null} allowCustom />
        </Wrapper>
      );
      const input = screen.getByLabelText('Backup');
      expect(input).not.toBeDisabled();
      expect(mocked.get).not.toHaveBeenCalled();

      await user.type(input, '/backups/mydumper/latest');
      await waitFor(() =>
        expect(screen.getByTestId('value')).toHaveTextContent(
          '"/backups/mydumper/latest"'
        )
      );
    });

    it('stays typable with free-text entry when the options fetch fails', async () => {
      mocked.get.mockRejectedValue(new Error('boom'));
      const user = userEvent.setup();
      render(
        <Wrapper client={makeClient()}>
          <Harness dependsOn="cluster" initialParent="cluster-a" allowCustom />
        </Wrapper>
      );
      await waitFor(() => expect(screen.getByText('boom')).toBeInTheDocument());

      const input = screen.getByLabelText('Backup');
      expect(input).not.toBeDisabled();
      await user.type(input, '/backups/mydumper/latest');
      await waitFor(() =>
        expect(screen.getByTestId('value')).toHaveTextContent(
          '"/backups/mydumper/latest"'
        )
      );
    });
  });

  describe('allow_custom', () => {
    it('commits a free-typed value verbatim as a string', async () => {
      const user = userEvent.setup();
      render(
        <Wrapper client={makeClient()}>
          <Harness allowCustom />
        </Wrapper>
      );
      await waitFor(() => expect(mocked.get).toHaveBeenCalled());

      await user.type(screen.getByRole('combobox'), 'my-custom-backup');
      await waitFor(() =>
        expect(screen.getByTestId('value')).toHaveTextContent(
          '"my-custom-backup"'
        )
      );
    });

    it('displays a persisted custom value with no matching option', async () => {
      render(
        <Wrapper client={makeClient()}>
          <Harness allowCustom initialBackup="ghost-value" />
        </Wrapper>
      );
      await waitFor(() => expect(mocked.get).toHaveBeenCalled());
      // free-solo keeps the unmatched string (contrast with the select-mode case).
      expect(screen.getByRole('combobox')).toHaveValue('ghost-value');
    });

    it('rejects typed whitespace-only input when required', async () => {
      const user = userEvent.setup();
      render(
        <Wrapper client={makeClient()}>
          <Harness allowCustom required />
        </Wrapper>
      );
      await waitFor(() => expect(mocked.get).toHaveBeenCalled());

      await user.type(screen.getByRole('combobox'), '   ');
      await user.click(screen.getByRole('button', { name: 'validate' }));
      await waitFor(() =>
        expect(screen.getByTestId('error')).toHaveTextContent(
          'Backup is required'
        )
      );
    });

    it('rejects a whitespace-only value loaded into the field when required', async () => {
      // normalizeChange collapses typed whitespace to null, so only a value
      // arriving from defaultValues/setValue reaches the validator's trim check.
      const user = userEvent.setup();
      render(
        <Wrapper client={makeClient()}>
          <Harness allowCustom required initialBackup="   " />
        </Wrapper>
      );
      await waitFor(() => expect(mocked.get).toHaveBeenCalled());

      await user.click(screen.getByRole('button', { name: 'validate' }));
      await waitFor(() =>
        expect(screen.getByTestId('error')).toHaveTextContent(
          'Backup is required'
        )
      );
    });

    it('keeps a disabled option label as free text across a parent change', async () => {
      // Typing a disabled option's label must not be classified as an option pick,
      // or the next cascade-parent change would wipe a value normalizeChange left
      // as verbatim free text.
      const user = userEvent.setup();
      render(
        <Wrapper client={makeClient()}>
          <Harness dependsOn="cluster" initialParent="cluster-a" allowCustom />
        </Wrapper>
      );
      await waitFor(() => expect(mocked.get).toHaveBeenCalled());

      await user.type(screen.getByRole('combobox'), 'Backup 2');
      await waitFor(() =>
        expect(screen.getByTestId('value')).toHaveTextContent('"Backup 2"')
      );

      await user.click(screen.getByRole('button', { name: 'change-parent' }));
      await waitFor(() =>
        expect(screen.getByTestId('value')).toHaveTextContent('"Backup 2"')
      );
    });
  });
});
