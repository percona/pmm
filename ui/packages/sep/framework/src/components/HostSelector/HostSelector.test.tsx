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
import { SnackbarProvider } from 'notistack';
import type { PropsWithChildren } from 'react';
import { HostSelector } from './HostSelector';
import { SchemaFormRenderer } from '../SchemaFormRenderer';
import type { FormSection } from '../SchemaFormRenderer/types';

vi.mock('@sep/api', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@sep/api')>()),
  apiClient: { get: vi.fn(), post: vi.fn() },
}));
import { ApiError, apiClient } from '@sep/api';
const mocked = apiClient as unknown as { get: ReturnType<typeof vi.fn> };

function makeResponse(
  items: Array<{ id: string; name: string; address: string }>
) {
  return { data: items };
}

function makeClient() {
  return new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0, staleTime: 0 } },
  });
}

function Harness() {
  const methods = useForm({ defaultValues: { hostId: null } });
  return (
    <FormProvider {...methods}>
      <HostSelector name="hostId" label="Host" />
    </FormProvider>
  );
}

function Wrapper({
  children,
  client,
}: PropsWithChildren<{ client: QueryClient }>) {
  return (
    <QueryClientProvider client={client}>
      <SnackbarProvider>{children}</SnackbarProvider>
    </QueryClientProvider>
  );
}

describe('HostSelector', () => {
  beforeEach(() => {
    mocked.get.mockReset();
  });

  it('fetches hosts via /api/sep/hosts/ and renders display names', async () => {
    mocked.get.mockResolvedValueOnce(
      makeResponse([
        { id: 'nomad-1', name: 'db-mysql-prod-01', address: '10.0.0.1' },
        { id: 'nomad-2', name: 'db-mysql-prod-02', address: '10.0.0.2' },
      ])
    );

    const client = makeClient();
    render(
      <Wrapper client={client}>
        <Harness />
      </Wrapper>
    );

    await waitFor(() => expect(mocked.get).toHaveBeenCalledWith('/sep/hosts/'));
    expect(mocked.get).toHaveBeenCalledTimes(1);

    const user = userEvent.setup();
    await user.click(screen.getByLabelText('Host'));
    expect(await screen.findByText('db-mysql-prod-01')).toBeInTheDocument();
    expect(screen.getByText('db-mysql-prod-02')).toBeInTheDocument();
  });

  it('renders empty state when the endpoint returns no hosts', async () => {
    mocked.get.mockResolvedValueOnce(makeResponse([]));
    const client = makeClient();
    render(
      <Wrapper client={client}>
        <Harness />
      </Wrapper>
    );
    // "No hosts available" appears in both the helperText and the
    // dropdown's noOptionsText slot, so assert via findAllByText.
    const matches = await screen.findAllByText('No hosts available');
    expect(matches.length).toBeGreaterThanOrEqual(1);
  });

  it('renders error state but stays enabled when the endpoint rejects, so opening retries', async () => {
    mocked.get.mockRejectedValueOnce(
      new ApiError({ kind: 'http', status: 502, message: 'boom' })
    );
    const client = makeClient();
    render(
      <Wrapper client={client}>
        <Harness />
      </Wrapper>
    );
    await screen.findByText('boom');
    const input = screen.getByLabelText('Host');
    // `onOpen` holds the only retry trigger, and a disabled Autocomplete never
    // opens, so disabling here would wedge the field until the page remounts.
    expect(input).not.toBeDisabled();

    mocked.get.mockResolvedValueOnce(
      makeResponse([
        { id: 'nomad-1', name: 'db-mysql-prod-01', address: '10.0.0.1' },
      ])
    );
    const user = userEvent.setup();
    await user.click(input);

    expect(await screen.findByText('db-mysql-prod-01')).toBeInTheDocument();
  });

  it('shows a loading message before the endpoint resolves', async () => {
    let resolveFetch!: (value: unknown) => void;
    mocked.get.mockReturnValueOnce(
      new Promise((resolve) => {
        resolveFetch = resolve;
      })
    );
    const client = makeClient();
    render(
      <Wrapper client={client}>
        <Harness />
      </Wrapper>
    );

    const user = userEvent.setup();
    await user.click(screen.getByLabelText('Host'));
    expect(await screen.findByText('Loading hosts…')).toBeInTheDocument();

    resolveFetch(makeResponse([]));
    // After resolution the empty-state text appears in both the helperText
    // and the dropdown's noOptionsText slot — match either.
    const matches = await screen.findAllByText('No hosts available');
    expect(matches.length).toBeGreaterThanOrEqual(1);
  });

  it('unwraps the selected option to the scalar id when submitting through SchemaFormRenderer', async () => {
    mocked.get.mockResolvedValue(
      makeResponse([
        { id: 'nomad-1', name: 'db-mysql-prod-01', address: '10.0.0.1' },
      ])
    );

    const onSubmit = vi.fn();
    const sections: FormSection[] = [
      {
        title: 'Target',
        fields: [
          { type: 'host', name: 'hostId', label: 'Host', required: true },
        ],
      },
    ];

    const client = makeClient();
    render(
      <Wrapper client={client}>
        <SchemaFormRenderer sections={sections} onSubmit={onSubmit} />
      </Wrapper>
    );

    await waitFor(() => expect(mocked.get).toHaveBeenCalledWith('/sep/hosts/'));

    const user = userEvent.setup();
    // `required: true` adds a trailing "*" to the rendered MUI label, so match by prefix.
    await user.click(screen.getByLabelText(/^Host\b/));
    const option = await screen.findByRole('option', {
      name: 'db-mysql-prod-01',
    });
    await user.click(option);

    await user.click(screen.getByRole('button', { name: /Run/ }));

    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
    expect(onSubmit).toHaveBeenCalledWith(
      expect.objectContaining({ hostId: 'nomad-1' })
    );
  });

  it('refetches /api/sep/hosts/ when the dropdown is opened', async () => {
    const hosts = [
      { id: 'nomad-1', name: 'db-mysql-prod-01', address: '10.0.0.1' },
    ];
    mocked.get.mockResolvedValue(makeResponse(hosts));

    const client = makeClient();
    render(
      <Wrapper client={client}>
        <Harness />
      </Wrapper>
    );

    await waitFor(() => expect(mocked.get).toHaveBeenCalledTimes(1));

    const user = userEvent.setup();
    await user.click(screen.getByLabelText('Host'));

    await waitFor(() => expect(mocked.get).toHaveBeenCalledTimes(2));
  });

  it('raises a snackbar when the hosts query fails with an upstream error', async () => {
    mocked.get.mockRejectedValueOnce(
      new ApiError({ kind: 'http', status: 502, message: 'tasks unreachable' })
    );

    const client = makeClient();
    render(
      <Wrapper client={client}>
        <Harness />
      </Wrapper>
    );

    expect(
      await screen.findByText(
        /Failed to load executor hosts: tasks unreachable/
      )
    ).toBeInTheDocument();
  });

  describe('allow_custom (free-solo)', () => {
    function CustomProbe() {
      const methods = useForm<{ host: unknown }>({
        defaultValues: { host: null },
      });
      return (
        <FormProvider {...methods}>
          <HostSelector name="host" label="Host" allowCustom />
          <output data-testid="host-value">
            {JSON.stringify(methods.watch('host'))}
          </output>
        </FormProvider>
      );
    }

    const value = () => screen.getByTestId('host-value').textContent;

    it('commits the host id when a host is picked', async () => {
      mocked.get.mockResolvedValueOnce(
        makeResponse([
          { id: 'nomad-1', name: 'db-mysql-prod-01', address: '10.0.0.1' },
          { id: 'nomad-2', name: 'db-mysql-prod-02', address: '10.0.0.2' },
        ])
      );
      const client = makeClient();
      const user = userEvent.setup();
      render(
        <Wrapper client={client}>
          <CustomProbe />
        </Wrapper>
      );
      await waitFor(() => expect(mocked.get).toHaveBeenCalled());
      await user.click(screen.getByLabelText('Host'));
      await user.click(await screen.findByText('db-mysql-prod-01'));
      expect(value()).toBe('"nomad-1"');
    });

    it('commits a typed value as a string', async () => {
      mocked.get.mockResolvedValueOnce(
        makeResponse([
          { id: 'nomad-1', name: 'db-mysql-prod-01', address: '10.0.0.1' },
        ])
      );
      const client = makeClient();
      const user = userEvent.setup();
      render(
        <Wrapper client={client}>
          <CustomProbe />
        </Wrapper>
      );
      await waitFor(() => expect(mocked.get).toHaveBeenCalled());
      await user.type(screen.getByLabelText('Host'), 'custom-executor');
      expect(value()).toBe('"custom-executor"');
    });

    it('back-compat: without allowCustom a typed value is not committed', async () => {
      mocked.get.mockResolvedValueOnce(
        makeResponse([
          { id: 'nomad-1', name: 'db-mysql-prod-01', address: '10.0.0.1' },
        ])
      );
      const client = makeClient();
      const user = userEvent.setup();
      function Probe() {
        const methods = useForm<{ host: unknown }>({
          defaultValues: { host: null },
        });
        return (
          <FormProvider {...methods}>
            <HostSelector name="host" label="Host" />
            <output data-testid="bc-value">
              {JSON.stringify(methods.watch('host'))}
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
      await user.type(screen.getByLabelText('Host'), 'custom-executor');
      expect(screen.getByTestId('bc-value').textContent).toBe('null');
    });

    it('unwraps a free-typed host through SchemaFormRenderer submit', async () => {
      mocked.get.mockResolvedValue(
        makeResponse([
          { id: 'nomad-1', name: 'db-mysql-prod-01', address: '10.0.0.1' },
        ])
      );

      const onSubmit = vi.fn();
      const sections: FormSection[] = [
        {
          title: 'Target',
          fields: [
            {
              type: 'host',
              name: 'hostId',
              label: 'Host',
              required: true,
              allow_custom: true,
            },
          ],
        },
      ];

      const client = makeClient();
      render(
        <Wrapper client={client}>
          <SchemaFormRenderer sections={sections} onSubmit={onSubmit} />
        </Wrapper>
      );

      await waitFor(() =>
        expect(mocked.get).toHaveBeenCalledWith('/sep/hosts/')
      );

      const user = userEvent.setup();
      await user.type(screen.getByLabelText(/^Host\b/), 'custom-executor');
      await user.click(screen.getByRole('button', { name: /Run/ }));

      await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
      expect(onSubmit).toHaveBeenCalledWith(
        expect.objectContaining({ hostId: 'custom-executor' })
      );
    });

    it('cascade auto-select commits a scalar host id when allowCustom is set', async () => {
      mocked.get.mockImplementation((url: string) => {
        if (url === '/sep/hosts/') {
          return Promise.resolve(
            makeResponse([
              { id: 'node-a', name: 'Display A', address: '10.0.0.1' },
              { id: 'node-b', name: 'Display B', address: '10.0.0.2' },
            ])
          );
        }
        if (url === '/sep/services/') {
          return Promise.resolve({
            data: {
              items: [
                {
                  id: 7,
                  name: 'mongo-svc',
                  type: 'mongodb',
                  node: {
                    name: 'node-b',
                    address: '10.0.0.2',
                    type: 'generic',
                  },
                  node_id: 1,
                  schemas: [],
                },
              ],
              total: 1,
              offset: 0,
              limit: 200,
            },
          });
        }
        return Promise.reject(new Error(`unexpected url ${url}`));
      });

      const onSubmit = vi.fn();
      const sections: FormSection[] = [
        {
          title: 'Task',
          fields: [
            {
              type: 'service',
              name: 'service_id',
              label: 'Database Service',
              required: true,
              service_types: ['mongodb'],
            },
            {
              type: 'host',
              name: 'hostname',
              label: 'Execution Host',
              required: true,
              depends_on: 'service_id',
              allow_custom: true,
            },
          ],
        },
      ];

      const client = makeClient();
      render(
        <Wrapper client={client}>
          <SchemaFormRenderer sections={sections} onSubmit={onSubmit} />
        </Wrapper>
      );

      await waitFor(() =>
        expect(mocked.get).toHaveBeenCalledWith('/sep/hosts/')
      );

      const user = userEvent.setup();
      await user.click(screen.getByLabelText(/^Database Service\b/));
      await user.click(
        await screen.findByRole('option', { name: 'mongo-svc (mongodb)' })
      );

      await waitFor(() => {
        expect(screen.getByLabelText(/^Execution Host\b/)).toHaveValue(
          'Display B'
        );
      });

      await user.click(screen.getByRole('button', { name: /Run/ }));
      await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
      expect(onSubmit).toHaveBeenCalledWith(
        expect.objectContaining({ hostname: 'node-b', service_id: 7 })
      );
    });

    it('refetches /api/sep/hosts/ when the free-solo dropdown is opened', async () => {
      mocked.get.mockResolvedValue(
        makeResponse([
          { id: 'nomad-1', name: 'db-mysql-prod-01', address: '10.0.0.1' },
        ])
      );

      const client = makeClient();
      render(
        <Wrapper client={client}>
          <CustomProbe />
        </Wrapper>
      );

      await waitFor(() => expect(mocked.get).toHaveBeenCalledTimes(1));

      const user = userEvent.setup();
      await user.click(screen.getByLabelText('Host'));

      await waitFor(() => expect(mocked.get).toHaveBeenCalledTimes(2));
    });
  });
});
