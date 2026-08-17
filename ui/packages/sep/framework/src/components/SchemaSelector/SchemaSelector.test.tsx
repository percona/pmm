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
import { SchemaSelector } from './SchemaSelector';
import type { ServiceOption } from '../../hooks/useServices';
import type { SchemaOption } from '../../hooks/useSchemas';

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
  service: ServiceOption | null;
  schema: SchemaOption | null;
}

function Harness({
  initialService,
  onMount,
}: {
  initialService: ServiceOption | null;
  onMount?: (api: { setService: (s: ServiceOption | null) => void }) => void;
}) {
  const methods = useForm<FormShape>({
    defaultValues: { service: initialService, schema: null },
  });
  if (onMount) {
    onMount({ setService: (s) => methods.setValue('service', s) });
  }
  return (
    <FormProvider {...methods}>
      <SchemaSelector name="schema" label="Schema" dependsOn="service" />
    </FormProvider>
  );
}

function Wrapper({
  children,
  client,
}: PropsWithChildren<{ client: QueryClient }>) {
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

describe('SchemaSelector', () => {
  beforeEach(() => {
    mocked.get.mockReset();
  });

  it('is disabled and skips fetch when no service selected', async () => {
    const client = makeClient();
    render(
      <Wrapper client={client}>
        <Harness initialService={null} />
      </Wrapper>
    );
    expect(screen.getByLabelText('Schema')).toBeDisabled();
    expect(screen.getByText('Select a service first')).toBeInTheDocument();
    expect(mocked.get).not.toHaveBeenCalled();
  });

  it('fetches schemas when service is set', async () => {
    mocked.get.mockResolvedValueOnce({
      data: [
        { id: 10, name: 'app_prod' },
        { id: 11, name: 'analytics' },
      ],
    });

    const client = makeClient();
    render(
      <Wrapper client={client}>
        <Harness initialService={{ id: 7, name: 'svc', type: 'mysql' }} />
      </Wrapper>
    );

    await waitFor(() =>
      expect(mocked.get).toHaveBeenCalledWith('/sep/services/7/schemas')
    );
  });

  it('resets value when parent service changes', async () => {
    mocked.get.mockResolvedValue({ data: [{ id: 1, name: 'a' }] });
    const client = makeClient();
    const apiRef: { current: ((s: ServiceOption | null) => void) | null } = {
      current: null,
    };
    const initial: ServiceOption = { id: 1, name: 's1', type: 'mysql' };

    function Probe() {
      const methods = useForm<FormShape>({
        defaultValues: {
          service: initial,
          schema: { id: 99, name: 'pre-existing' },
        },
      });
      apiRef.current = (s) => methods.setValue('service', s);
      return (
        <FormProvider {...methods}>
          <SchemaSelector name="schema" label="Schema" dependsOn="service" />
          <output data-testid="schema-value">
            {JSON.stringify(methods.watch('schema'))}
          </output>
        </FormProvider>
      );
    }

    render(
      <Wrapper client={client}>
        <Probe />
      </Wrapper>
    );

    expect(screen.getByTestId('schema-value').textContent).toContain(
      'pre-existing'
    );

    await act(async () => {
      apiRef.current?.({ id: 2, name: 's2', type: 'mysql' });
    });

    await waitFor(() => {
      expect(screen.getByTestId('schema-value').textContent).toBe('null');
    });
  });

  it('renders error state', async () => {
    mocked.get.mockRejectedValueOnce(new Error('schemas down'));
    const client = makeClient();
    render(
      <Wrapper client={client}>
        <Harness initialService={{ id: 1, name: 's', type: 'mysql' }} />
      </Wrapper>
    );
    await screen.findByText('schemas down');
  });

  it('renders empty state', async () => {
    mocked.get.mockResolvedValueOnce({ data: [] });
    const client = makeClient();
    render(
      <Wrapper client={client}>
        <Harness initialService={{ id: 1, name: 's', type: 'mysql' }} />
      </Wrapper>
    );
    await screen.findByText('No schemas in this service');
  });

  describe('allow_custom (free-solo)', () => {
    // The allow_custom path commits `number | string | null`, not a `SchemaOption`.
    interface CustomFormShape {
      service: ServiceOption | null;
      schema: number | string | null;
    }

    function CustomProbe() {
      const methods = useForm<CustomFormShape>({
        defaultValues: {
          service: { id: 7, name: 'svc', type: 'mysql' },
          schema: null,
        },
      });
      return (
        <FormProvider {...methods}>
          <SchemaSelector
            name="schema"
            label="Schema"
            dependsOn="service"
            allowCustom
          />
          <output data-testid="schema-value">
            {JSON.stringify(methods.watch('schema'))}
          </output>
        </FormProvider>
      );
    }

    const value = () => screen.getByTestId('schema-value').textContent;

    it('commits the inventory id when a schema is picked', async () => {
      mocked.get.mockResolvedValue({
        data: [
          { id: 10, name: 'app_prod' },
          { id: 11, name: 'analytics' },
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
      await user.click(screen.getByLabelText('Schema'));
      await user.click(await screen.findByText('app_prod'));
      expect(value()).toBe('10');
    });

    it('commits a typed value as a string', async () => {
      mocked.get.mockResolvedValue({ data: [{ id: 10, name: 'app_prod' }] });
      const client = makeClient();
      const user = userEvent.setup();
      render(
        <Wrapper client={client}>
          <CustomProbe />
        </Wrapper>
      );
      await waitFor(() => expect(mocked.get).toHaveBeenCalled());
      await user.type(screen.getByLabelText('Schema'), 'manual_db');
      expect(value()).toBe('"manual_db"');
    });

    it('stays enabled and accepts a typed value when the parent service is custom', async () => {
      const client = makeClient();
      const user = userEvent.setup();
      function CascadeProbe() {
        // Parent service holds a free-typed (custom) string, not an inventory id.
        const methods = useForm<{ service: unknown; schema: unknown }>({
          defaultValues: { service: 'custom-svc', schema: null },
        });
        return (
          <FormProvider {...methods}>
            <SchemaSelector
              name="schema"
              label="Schema"
              dependsOn="service"
              allowCustom
            />
            <output data-testid="schema-value">
              {JSON.stringify(methods.watch('schema'))}
            </output>
          </FormProvider>
        );
      }
      render(
        <Wrapper client={client}>
          <CascadeProbe />
        </Wrapper>
      );
      const input = screen.getByLabelText('Schema');
      expect(input).not.toBeDisabled();
      // No service id means no schema fetch is issued.
      expect(mocked.get).not.toHaveBeenCalled();
      await user.type(input, 'custom_schema');
      expect(screen.getByTestId('schema-value').textContent).toBe(
        '"custom_schema"'
      );
    });

    it('treats a numeric parent string as an inventory id and loads its schemas', async () => {
      // On edit, reference ids are persisted as strings (e.g. `"42"`); the
      // dependent selector must resolve them to inventory ids and fetch, not
      // mistake them for a free-typed custom host.
      mocked.get.mockResolvedValue({
        data: [
          { id: 10, name: 'app_prod' },
          { id: 11, name: 'analytics' },
        ],
      });
      const client = makeClient();
      function NumericParentProbe() {
        const methods = useForm<{ service: unknown; schema: unknown }>({
          defaultValues: { service: '42', schema: null },
        });
        return (
          <FormProvider {...methods}>
            <SchemaSelector
              name="schema"
              label="Schema"
              dependsOn="service"
              allowCustom
            />
            <output data-testid="schema-value">
              {JSON.stringify(methods.watch('schema'))}
            </output>
          </FormProvider>
        );
      }
      render(
        <Wrapper client={client}>
          <NumericParentProbe />
        </Wrapper>
      );
      expect(screen.getByLabelText('Schema')).not.toBeDisabled();
      await waitFor(() =>
        expect(mocked.get).toHaveBeenCalledWith('/sep/services/42/schemas')
      );
    });

    it('resolves a stringified child schema id to its option name on edit', async () => {
      // Reproduces the edit-form bug: both parent (service) and child (schema)
      // come back as stringified ids; the child must render as the schema name,
      // not the raw id, and must not be wiped by the parent-change reset.
      mocked.get.mockResolvedValue({
        data: [
          { id: 10, name: 'app_prod' },
          { id: 11, name: 'analytics' },
        ],
      });
      const client = makeClient();
      function EditProbe() {
        const methods = useForm<{ service: unknown; schema: unknown }>({
          defaultValues: { service: '7', schema: '11' },
        });
        return (
          <FormProvider {...methods}>
            <SchemaSelector
              name="schema"
              label="Schema"
              dependsOn="service"
              allowCustom
            />
            <output data-testid="schema-value">
              {JSON.stringify(methods.watch('schema'))}
            </output>
          </FormProvider>
        );
      }
      render(
        <Wrapper client={client}>
          <EditProbe />
        </Wrapper>
      );
      await waitFor(() =>
        expect(screen.getByLabelText('Schema')).toHaveValue('analytics')
      );
      expect(screen.getByTestId('schema-value').textContent).toBe('11');
    });

    it('clears child value when parent custom value changes', async () => {
      const client = makeClient();
      const setServiceRef: { current: ((value: unknown) => void) | null } = {
        current: null,
      };
      function CustomParentChangeProbe() {
        const methods = useForm<{ service: unknown; schema: unknown }>({
          defaultValues: { service: 'custom-svc-a', schema: 'child-schema' },
        });
        setServiceRef.current = (value) => methods.setValue('service', value);
        return (
          <FormProvider {...methods}>
            <SchemaSelector
              name="schema"
              label="Schema"
              dependsOn="service"
              allowCustom
            />
            <output data-testid="schema-value">
              {JSON.stringify(methods.watch('schema'))}
            </output>
          </FormProvider>
        );
      }
      render(
        <Wrapper client={client}>
          <CustomParentChangeProbe />
        </Wrapper>
      );
      expect(screen.getByTestId('schema-value').textContent).toBe(
        '"child-schema"'
      );
      await act(async () => {
        setServiceRef.current?.('custom-svc-b');
      });
      await waitFor(() => {
        expect(screen.getByTestId('schema-value').textContent).toBe('null');
      });
      expect(mocked.get).not.toHaveBeenCalled();
    });
  });

  it('back-compat: without allowCustom a typed value is not committed', async () => {
    mocked.get.mockResolvedValue({ data: [{ id: 10, name: 'app_prod' }] });
    const client = makeClient();
    const user = userEvent.setup();
    function Probe() {
      const methods = useForm<FormShape>({
        defaultValues: {
          service: { id: 7, name: 'svc', type: 'mysql' },
          schema: null,
        },
      });
      return (
        <FormProvider {...methods}>
          <SchemaSelector name="schema" label="Schema" dependsOn="service" />
          <output data-testid="bc-value">
            {JSON.stringify(methods.watch('schema'))}
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
    await user.type(screen.getByLabelText('Schema'), 'manual_db');
    expect(screen.getByTestId('bc-value').textContent).toBe('null');
  });
});
