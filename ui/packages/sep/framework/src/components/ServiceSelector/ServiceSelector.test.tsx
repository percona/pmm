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
import { ServiceSelector } from './ServiceSelector';

vi.mock('@sep/api', async (importOriginal) => ({
  // Keep real exports (notably ``ApiError``) so ``sepRetry``'s
  // ``err instanceof ApiError`` check inside ``useServices`` resolves
  // when this test exercises the error path.
  ...(await importOriginal<typeof import('@sep/api')>()),
  apiClient: { get: vi.fn(), post: vi.fn() },
}));
import { apiClient } from '@sep/api';
const mocked = apiClient as unknown as { get: ReturnType<typeof vi.fn> };

function makePage(items: Array<{ id: number; name: string; type: string }>) {
  return { data: { items, total: items.length, offset: 0, limit: 200 } };
}

function makeClient() {
  // ``useServices`` sets ``retry: sepRetry`` at the query level, which
  // overrides the client-level ``retry: false`` default. ``retryDelay: 0``
  // collapses the exponential backoff so error-path tests finish promptly
  // regardless of the retry policy.
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false, retryDelay: 0, gcTime: 0, staleTime: 0 },
    },
  });
}

function Harness({ serviceTypes }: { serviceTypes?: readonly string[] }) {
  const methods = useForm({ defaultValues: { service: null } });
  return (
    <FormProvider {...methods}>
      <ServiceSelector
        name="service"
        label="Service"
        serviceTypes={serviceTypes as never}
      />
    </FormProvider>
  );
}

function Wrapper({
  children,
  client,
}: PropsWithChildren<{ client: QueryClient }>) {
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

describe('ServiceSelector', () => {
  beforeEach(() => {
    mocked.get.mockReset();
  });

  it('fetches services and renders options', async () => {
    mocked.get.mockResolvedValueOnce(
      makePage([
        { id: 1, name: 'mysql-prod', type: 'mysql' },
        { id: 2, name: 'pg-prod', type: 'postgresql' },
      ])
    );

    const client = makeClient();
    render(
      <Wrapper client={client}>
        <Harness />
      </Wrapper>
    );

    await waitFor(() =>
      expect(mocked.get).toHaveBeenCalledWith('/sep/services/', {
        params: { offset: 0, limit: 200 },
      })
    );

    const user = userEvent.setup();
    await user.click(screen.getByLabelText('Service'));
    expect(await screen.findByText('mysql-prod (mysql)')).toBeInTheDocument();
    expect(screen.getByText('pg-prod (postgresql)')).toBeInTheDocument();
  });

  it('passes serviceTypes filter as query param', async () => {
    mocked.get.mockResolvedValueOnce(
      makePage([{ id: 1, name: 'mysql-prod', type: 'mysql' }])
    );

    const client = makeClient();
    render(
      <Wrapper client={client}>
        <Harness serviceTypes={['mysql']} />
      </Wrapper>
    );

    await waitFor(() =>
      expect(mocked.get).toHaveBeenCalledWith('/sep/services/', {
        params: { offset: 0, limit: 200, service_type: 'mysql' },
      })
    );
  });

  it('renders error state', async () => {
    // Reject every attempt — sepRetry retries plain Errors up to 2 times.
    mocked.get.mockRejectedValue(new Error('boom'));
    const client = makeClient();
    render(
      <Wrapper client={client}>
        <Harness />
      </Wrapper>
    );
    await screen.findByText('boom');
  });

  it('renders empty state', async () => {
    mocked.get.mockResolvedValueOnce(makePage([]));
    const client = makeClient();
    render(
      <Wrapper client={client}>
        <Harness />
      </Wrapper>
    );
    await screen.findByText('No services available');
  });

  it('shows the required error message when submitted empty', async () => {
    mocked.get.mockResolvedValueOnce(
      makePage([{ id: 1, name: 'mysql-prod', type: 'mysql' }])
    );
    const client = makeClient();
    const onSubmit = vi.fn();
    function RequiredProbe() {
      const methods = useForm({ defaultValues: { service: null } });
      return (
        <FormProvider {...methods}>
          <form onSubmit={methods.handleSubmit(onSubmit)} noValidate>
            <ServiceSelector name="service" label="Service" required />
            <button type="submit">Submit</button>
          </form>
        </FormProvider>
      );
    }
    const user = userEvent.setup();
    render(
      <Wrapper client={client}>
        <RequiredProbe />
      </Wrapper>
    );
    await waitFor(() => expect(mocked.get).toHaveBeenCalled());
    await user.click(screen.getByRole('button', { name: 'Submit' }));
    expect(await screen.findByText('Service is required')).toBeInTheDocument();
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it('hydrates a persisted scalar service id into the matching inventory option', async () => {
    mocked.get.mockResolvedValueOnce(
      makePage([{ id: 7, name: 'mysql-prod', type: 'mysql' }])
    );
    const client = makeClient();
    function Probe() {
      const methods = useForm<{ service: unknown }>({
        defaultValues: { service: 7 },
      });
      return (
        <FormProvider {...methods}>
          <ServiceSelector
            name="service"
            label="Service"
            serviceTypes={['mysql']}
          />
          <output data-testid="hydrated-value">
            {JSON.stringify(methods.watch('service'))}
          </output>
        </FormProvider>
      );
    }
    render(
      <Wrapper client={client}>
        <Probe />
      </Wrapper>
    );
    await waitFor(() =>
      expect(screen.getByTestId('hydrated-value').textContent).toContain(
        '"id":7'
      )
    );
    expect(screen.getByLabelText('Service')).toHaveValue('mysql-prod (mysql)');
  });

  describe('allow_custom (free-solo)', () => {
    function CustomProbe() {
      const methods = useForm<{ service: unknown }>({
        defaultValues: { service: null },
      });
      return (
        <FormProvider {...methods}>
          <ServiceSelector name="service" label="Service" allowCustom />
          <output data-testid="service-value">
            {JSON.stringify(methods.watch('service'))}
          </output>
        </FormProvider>
      );
    }

    const value = () => screen.getByTestId('service-value').textContent;

    it('commits the inventory id when a service is picked', async () => {
      mocked.get.mockResolvedValueOnce(
        makePage([
          { id: 1, name: 'mysql-prod', type: 'mysql' },
          { id: 2, name: 'pg-prod', type: 'postgresql' },
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
      await user.click(screen.getByLabelText('Service'));
      await user.click(await screen.findByText('mysql-prod (mysql)'));
      expect(value()).toBe('1');
    });

    it('commits a typed value as a string', async () => {
      mocked.get.mockResolvedValueOnce(
        makePage([{ id: 1, name: 'mysql-prod', type: 'mysql' }])
      );
      const client = makeClient();
      const user = userEvent.setup();
      render(
        <Wrapper client={client}>
          <CustomProbe />
        </Wrapper>
      );
      await waitFor(() => expect(mocked.get).toHaveBeenCalled());
      await user.type(screen.getByLabelText('Service'), 'external-svc');
      expect(value()).toBe('"external-svc"');
    });

    it('back-compat: without allowCustom a typed value is not committed', async () => {
      mocked.get.mockResolvedValueOnce(
        makePage([{ id: 1, name: 'mysql-prod', type: 'mysql' }])
      );
      const client = makeClient();
      const user = userEvent.setup();
      function Probe() {
        const methods = useForm<{ service: unknown }>({
          defaultValues: { service: null },
        });
        return (
          <FormProvider {...methods}>
            <ServiceSelector name="service" label="Service" />
            <output data-testid="bc-value">
              {JSON.stringify(methods.watch('service'))}
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
      await user.type(screen.getByLabelText('Service'), 'external-svc');
      expect(screen.getByTestId('bc-value').textContent).toBe('null');
    });
  });
});
