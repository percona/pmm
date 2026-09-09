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

import { act, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { FormProvider, useForm } from 'react-hook-form';
import type { PropsWithChildren } from 'react';
import { TableSelector } from './TableSelector';
import type { SchemaOption } from '../../hooks/useSchemas';
import type { TableOption } from '../../hooks/useTables';

vi.mock('@sep/api', () => ({
  apiClient: { get: vi.fn(), post: vi.fn() },
}));
import { apiClient } from '@sep/api';
const mocked = apiClient as unknown as { get: ReturnType<typeof vi.fn> };

function makeClient() {
  return new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0, staleTime: 0 } },
  });
}

interface FormShape {
  schema: SchemaOption | null;
  table: TableOption | null;
}

function Wrapper({
  children,
  client,
}: PropsWithChildren<{ client: QueryClient }>) {
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

function Harness({ initialSchema }: { initialSchema: SchemaOption | null }) {
  const methods = useForm<FormShape>({
    defaultValues: { schema: initialSchema, table: null },
  });
  return (
    <FormProvider {...methods}>
      <TableSelector name="table" label="Table" dependsOn="schema" />
    </FormProvider>
  );
}

describe('TableSelector', () => {
  beforeEach(() => {
    mocked.get.mockReset();
  });

  it('disabled with no schema', () => {
    const client = makeClient();
    render(
      <Wrapper client={client}>
        <Harness initialSchema={null} />
      </Wrapper>
    );
    expect(screen.getByLabelText('Table')).toBeDisabled();
    expect(screen.getByText('Select a schema first')).toBeInTheDocument();
    expect(mocked.get).not.toHaveBeenCalled();
  });

  it('fetches tables when schema set', async () => {
    mocked.get.mockResolvedValueOnce({
      data: [
        { id: 100, name: 'users' },
        { id: 101, name: 'orders' },
      ],
    });
    const client = makeClient();
    render(
      <Wrapper client={client}>
        <Harness initialSchema={{ id: 42, name: 'app_prod' }} />
      </Wrapper>
    );
    await waitFor(() =>
      expect(mocked.get).toHaveBeenCalledWith('/sep/schemas/42/tables')
    );
  });

  it('resets value when parent schema changes', async () => {
    mocked.get.mockResolvedValue({ data: [{ id: 1, name: 't' }] });
    const client = makeClient();
    let setSchema!: (s: SchemaOption | null) => void;

    function Probe() {
      const methods = useForm<FormShape>({
        defaultValues: {
          schema: { id: 1, name: 's' },
          table: { id: 9, name: 'pre' },
        },
      });
      setSchema = (s) => methods.setValue('schema', s);
      return (
        <FormProvider {...methods}>
          <TableSelector name="table" label="Table" dependsOn="schema" />
          <output data-testid="table-value">
            {JSON.stringify(methods.watch('table'))}
          </output>
        </FormProvider>
      );
    }

    render(
      <Wrapper client={client}>
        <Probe />
      </Wrapper>
    );

    expect(screen.getByTestId('table-value').textContent).toContain('pre');

    await act(async () => {
      setSchema({ id: 2, name: 's2' });
    });

    await waitFor(() => {
      expect(screen.getByTestId('table-value').textContent).toBe('null');
    });
  });

  it('renders error state', async () => {
    mocked.get.mockRejectedValueOnce(new Error('tables down'));
    const client = makeClient();
    render(
      <Wrapper client={client}>
        <Harness initialSchema={{ id: 1, name: 's' }} />
      </Wrapper>
    );
    await screen.findByText('tables down');
  });

  it('renders empty state', async () => {
    mocked.get.mockResolvedValueOnce({ data: [] });
    const client = makeClient();
    render(
      <Wrapper client={client}>
        <Harness initialSchema={{ id: 1, name: 's' }} />
      </Wrapper>
    );
    await screen.findByText('No tables in this schema');
  });

  describe('allow_custom (free-solo)', () => {
    // The allow_custom path commits `number | string | null`, not a `TableOption`.
    interface CustomFormShape {
      schema: SchemaOption | null;
      table: number | string | null;
    }

    function CustomProbe() {
      const methods = useForm<CustomFormShape>({
        defaultValues: { schema: { id: 42, name: 'app_prod' }, table: null },
      });
      return (
        <FormProvider {...methods}>
          <TableSelector
            name="table"
            label="Table"
            dependsOn="schema"
            allowCustom
          />
          <output data-testid="table-value">
            {JSON.stringify(methods.watch('table'))}
          </output>
        </FormProvider>
      );
    }

    const value = () => screen.getByTestId('table-value').textContent;

    it('commits the inventory id when a table is picked', async () => {
      mocked.get.mockResolvedValue({
        data: [
          { id: 100, name: 'users' },
          { id: 101, name: 'orders' },
        ],
      });
      const client = makeClient();
      const user = userEvent.setup();
      render(
        <Wrapper client={client}>
          <CustomProbe />
        </Wrapper>
      );
      await waitFor(() => expect(mocked.get).toHaveBeenCalled());
      await user.click(screen.getByLabelText('Table'));
      await user.click(await screen.findByText('orders'));
      expect(value()).toBe('101');
    });

    it('commits a typed value as a string', async () => {
      mocked.get.mockResolvedValue({ data: [{ id: 100, name: 'users' }] });
      const client = makeClient();
      const user = userEvent.setup();
      render(
        <Wrapper client={client}>
          <CustomProbe />
        </Wrapper>
      );
      await waitFor(() => expect(mocked.get).toHaveBeenCalled());
      await user.type(screen.getByLabelText('Table'), 'manual_table');
      expect(value()).toBe('"manual_table"');
    });

    it('resolves a typed value matching an existing table to its id', async () => {
      mocked.get.mockResolvedValue({ data: [{ id: 100, name: 'users' }] });
      const client = makeClient();
      const user = userEvent.setup();
      render(
        <Wrapper client={client}>
          <CustomProbe />
        </Wrapper>
      );
      await waitFor(() => expect(mocked.get).toHaveBeenCalled());
      await user.type(screen.getByLabelText('Table'), 'users');
      expect(value()).toBe('100');
    });

    it('stays enabled and accepts a typed value when the parent schema is custom', async () => {
      const client = makeClient();
      const user = userEvent.setup();
      function CascadeProbe() {
        // Parent schema holds a free-typed (custom) string, not an inventory id.
        const methods = useForm<{ schema: unknown; table: unknown }>({
          defaultValues: { schema: 'custom_schema', table: null },
        });
        return (
          <FormProvider {...methods}>
            <TableSelector
              name="table"
              label="Table"
              dependsOn="schema"
              allowCustom
            />
            <output data-testid="table-value">
              {JSON.stringify(methods.watch('table'))}
            </output>
          </FormProvider>
        );
      }
      render(
        <Wrapper client={client}>
          <CascadeProbe />
        </Wrapper>
      );
      const input = screen.getByLabelText('Table');
      expect(input).not.toBeDisabled();
      // No schema id means no table fetch is issued.
      expect(mocked.get).not.toHaveBeenCalled();
      await user.type(input, 'custom_table');
      expect(screen.getByTestId('table-value').textContent).toBe(
        '"custom_table"'
      );
    });

    it('treats a numeric custom parent string as custom (not an inventory id)', async () => {
      const client = makeClient();
      const user = userEvent.setup();
      function NumericCustomParentProbe() {
        const methods = useForm<{ schema: unknown; table: unknown }>({
          defaultValues: { schema: '42', table: null },
        });
        return (
          <FormProvider {...methods}>
            <TableSelector
              name="table"
              label="Table"
              dependsOn="schema"
              allowCustom
            />
            <output data-testid="table-value">
              {JSON.stringify(methods.watch('table'))}
            </output>
          </FormProvider>
        );
      }
      render(
        <Wrapper client={client}>
          <NumericCustomParentProbe />
        </Wrapper>
      );
      const input = screen.getByLabelText('Table');
      expect(input).not.toBeDisabled();
      expect(mocked.get).not.toHaveBeenCalled();
      await user.type(input, 'custom_table');
      expect(screen.getByTestId('table-value').textContent).toBe(
        '"custom_table"'
      );
    });

    it('clears child value when parent custom value changes', async () => {
      const client = makeClient();
      const setSchemaRef: { current: ((value: unknown) => void) | null } = {
        current: null,
      };
      function CustomParentChangeProbe() {
        const methods = useForm<{ schema: unknown; table: unknown }>({
          defaultValues: { schema: 'custom-schema-a', table: 'child-table' },
        });
        setSchemaRef.current = (value) => methods.setValue('schema', value);
        return (
          <FormProvider {...methods}>
            <TableSelector
              name="table"
              label="Table"
              dependsOn="schema"
              allowCustom
            />
            <output data-testid="table-value">
              {JSON.stringify(methods.watch('table'))}
            </output>
          </FormProvider>
        );
      }
      render(
        <Wrapper client={client}>
          <CustomParentChangeProbe />
        </Wrapper>
      );
      expect(screen.getByTestId('table-value').textContent).toBe(
        '"child-table"'
      );
      await act(async () => {
        setSchemaRef.current?.('custom-schema-b');
      });
      await waitFor(() => {
        expect(screen.getByTestId('table-value').textContent).toBe('null');
      });
      expect(mocked.get).not.toHaveBeenCalled();
    });
  });

  it('back-compat: without allowCustom a typed value is not committed', async () => {
    mocked.get.mockResolvedValue({ data: [{ id: 100, name: 'users' }] });
    const client = makeClient();
    const user = userEvent.setup();
    function Probe() {
      const methods = useForm<FormShape>({
        defaultValues: { schema: { id: 42, name: 'app_prod' }, table: null },
      });
      return (
        <FormProvider {...methods}>
          <TableSelector name="table" label="Table" dependsOn="schema" />
          <output data-testid="bc-value">
            {JSON.stringify(methods.watch('table'))}
          </output>
        </FormProvider>
      );
    }
    render(
      <Wrapper client={client}>
        <Probe />
      </Wrapper>
    );
    await waitFor(() => expect(mocked.get).toHaveBeenCalled());
    await user.type(screen.getByLabelText('Table'), 'manual_table');
    expect(screen.getByTestId('bc-value').textContent).toBe('null');
  });
});
