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
import { SnackbarProvider } from 'notistack';
import type { PropsWithChildren } from 'react';
import { StandaloneHostSelector } from './StandaloneHostSelector';

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

describe('StandaloneHostSelector', () => {
  beforeEach(() => {
    mocked.get.mockReset();
  });

  it('renders the label and calls onChange with host id on selection', async () => {
    mocked.get.mockResolvedValueOnce(
      makeResponse([
        { id: 'nomad-1', name: 'db-mysql-prod-01', address: '10.0.0.1' },
        { id: 'nomad-2', name: 'db-mysql-prod-02', address: '10.0.0.2' },
      ])
    );

    const handleChange = vi.fn();
    const client = makeClient();
    render(
      <Wrapper client={client}>
        <StandaloneHostSelector value="" onChange={handleChange} />
      </Wrapper>
    );

    const user = userEvent.setup();
    await user.click(screen.getByLabelText('Execution Host'));
    await user.click(
      await screen.findByRole('option', { name: 'db-mysql-prod-01' })
    );

    expect(handleChange).toHaveBeenCalledWith('nomad-1');
  });

  it('clears the selection by calling onChange with empty string when cleared', async () => {
    mocked.get.mockResolvedValueOnce(
      makeResponse([
        { id: 'nomad-1', name: 'db-mysql-prod-01', address: '10.0.0.1' },
      ])
    );

    const handleChange = vi.fn();
    const client = makeClient();
    render(
      <Wrapper client={client}>
        <StandaloneHostSelector value="nomad-1" onChange={handleChange} />
      </Wrapper>
    );

    await waitFor(() => expect(mocked.get).toHaveBeenCalled());

    const user = userEvent.setup();
    const clearButton = await screen.findByTitle('Clear');
    await user.click(clearButton);

    expect(handleChange).toHaveBeenCalledWith('');
  });

  it('disables the input and shows error text when the endpoint rejects', async () => {
    mocked.get.mockRejectedValueOnce(
      new ApiError({ kind: 'http', status: 502, message: 'network error' })
    );
    const client = makeClient();
    render(
      <Wrapper client={client}>
        <StandaloneHostSelector value="" onChange={vi.fn()} />
      </Wrapper>
    );

    await screen.findByText('network error');
    expect(screen.getByLabelText('Execution Host')).toBeDisabled();
  });

  it('surfaces upstream Tasks-API failure via snackbar', async () => {
    mocked.get.mockRejectedValueOnce(
      new ApiError({ kind: 'http', status: 502, message: 'tasks unreachable' })
    );

    const client = makeClient();
    render(
      <Wrapper client={client}>
        <StandaloneHostSelector value="" onChange={vi.fn()} />
      </Wrapper>
    );

    expect(
      await screen.findByText(
        /Failed to load executor hosts: tasks unreachable/
      )
    ).toBeInTheDocument();
  });

  it('accepts a custom label', async () => {
    mocked.get.mockResolvedValueOnce(makeResponse([]));
    const client = makeClient();
    render(
      <Wrapper client={client}>
        <StandaloneHostSelector
          value=""
          onChange={vi.fn()}
          label="Target Host"
        />
      </Wrapper>
    );

    expect(screen.getByLabelText('Target Host')).toBeInTheDocument();
  });

  it('shows "No hosts available" when the host list is empty', async () => {
    mocked.get.mockResolvedValueOnce(makeResponse([]));
    const client = makeClient();
    const user = userEvent.setup();
    render(
      <Wrapper client={client}>
        <StandaloneHostSelector value="" onChange={vi.fn()} />
      </Wrapper>
    );

    await user.click(screen.getByLabelText('Execution Host'));
    expect(await screen.findByText('No hosts available')).toBeInTheDocument();
  });

  it('refetches /api/sep/hosts/ when the dropdown is opened', async () => {
    const hosts = [
      { id: 'nomad-1', name: 'db-mysql-prod-01', address: '10.0.0.1' },
    ];
    mocked.get.mockResolvedValue(makeResponse(hosts));

    const client = makeClient();
    render(
      <Wrapper client={client}>
        <StandaloneHostSelector value="" onChange={vi.fn()} />
      </Wrapper>
    );

    await waitFor(() => expect(mocked.get).toHaveBeenCalledTimes(1));

    const user = userEvent.setup();
    await user.click(screen.getByLabelText('Execution Host'));

    await waitFor(() => expect(mocked.get).toHaveBeenCalledTimes(2));
  });

  it('shows "Loading hosts…" while the host list is loading', async () => {
    mocked.get.mockReturnValueOnce(new Promise(() => {}));
    const client = makeClient();
    const user = userEvent.setup();
    render(
      <Wrapper client={client}>
        <StandaloneHostSelector value="" onChange={vi.fn()} />
      </Wrapper>
    );

    await user.click(screen.getByLabelText('Execution Host'));
    expect(await screen.findByText('Loading hosts…')).toBeInTheDocument();
  });
});
