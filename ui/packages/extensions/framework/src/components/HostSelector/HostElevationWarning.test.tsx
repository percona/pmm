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
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { FormProvider, useForm } from 'react-hook-form';
import {
  HostElevationWarning,
  snippetsLaunchedWithSudo,
  type SnippetElevation,
} from './HostElevationWarning';

vi.mock('@pmm-extensions/api', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@pmm-extensions/api')>()),
  apiClient: { get: vi.fn(), post: vi.fn() },
}));
import { apiClient } from '@pmm-extensions/api';
const mocked = apiClient as unknown as { get: ReturnType<typeof vi.fn> };

const ALWAYS: SnippetElevation = { title: 'always.sh', sudo: 'always' };
const OPTIONAL: SnippetElevation = { title: 'optional.sh', sudo: 'optional' };
const NEVER: SnippetElevation = { title: 'never.sh', sudo: 'never' };
const UNDECLARED: SnippetElevation = { title: 'undeclared.sh' };

describe('snippetsLaunchedWithSudo', () => {
  it('keeps always-sudo snippets whatever the sudo choice', () => {
    expect(snippetsLaunchedWithSudo([ALWAYS, NEVER], false)).toEqual([ALWAYS]);
  });

  it('keeps optional-sudo snippets only when sudo is chosen', () => {
    expect(snippetsLaunchedWithSudo([OPTIONAL], false)).toEqual([]);
    expect(snippetsLaunchedWithSudo([OPTIONAL], true)).toEqual([OPTIONAL]);
  });

  it('never keeps a snippet that never elevates or does not say', () => {
    expect(snippetsLaunchedWithSudo([NEVER, UNDECLARED], true)).toEqual([]);
  });
});

function Harness({
  host,
  snippets,
}: {
  host: unknown;
  snippets: SnippetElevation[];
}) {
  const methods = useForm({ defaultValues: { host, sudo: false } });
  return (
    <FormProvider {...methods}>
      <HostElevationWarning name="host" sudoName="sudo" snippets={snippets} />
    </FormProvider>
  );
}

function renderWarning(host: unknown, snippets: SnippetElevation[]) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  render(
    <QueryClientProvider client={client}>
      <Harness host={host} snippets={snippets} />
    </QueryClientProvider>
  );
  return client;
}

describe('HostElevationWarning', () => {
  beforeEach(() => {
    mocked.get.mockReset();
    mocked.get.mockResolvedValue({
      data: [
        {
          id: 'nomad-1',
          name: 'db-01',
          address: '10.0.0.1',
          can_elevate: false,
        },
      ],
    });
  });

  it('reads a host held as a bare executor id, as a free-solo field stores it', async () => {
    renderWarning('nomad-1', [ALWAYS]);

    expect(
      await screen.findByText(/db-01 cannot run commands with sudo/)
    ).toHaveTextContent('always.sh');
  });

  it('stays silent when the server does not say whether a snippet elevates', async () => {
    const client = renderWarning('nomad-1', [UNDECLARED]);

    await waitFor(() => {
      expect(client.getQueryData(['extensions', 'hosts'])).toBeDefined();
    });
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
  });
});
