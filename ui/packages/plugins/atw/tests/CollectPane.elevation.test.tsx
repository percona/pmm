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
import type { ReactNode } from 'react';
import { CollectPane } from '../src/CollectPane';
import type { AtwSnippetSummary } from '../src/types';

vi.mock('@pmm-extensions/api', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@pmm-extensions/api')>()),
  apiClient: { get: vi.fn(), post: vi.fn() },
  useAuth: () => ({ isAdmin: true, canMutate: true }),
}));

import { apiClient } from '@pmm-extensions/api';
const mockedApi = apiClient as unknown as { get: ReturnType<typeof vi.fn> };

const PT_STALK: AtwSnippetSummary = {
  name: 'pt-stalk.sh',
  title: 'pt-stalk',
  description: 'Executes pt-stalk command.',
  sudo: 'always',
};

const DISK_USAGE: AtwSnippetSummary = {
  name: 'disk_usage.sh',
  title: 'Disk usage',
  description: 'Reports disk usage.',
  sudo: 'optional',
};

const PT_MYSQL_SUMMARY: AtwSnippetSummary = {
  name: 'pt-mysql-summary.sh',
  title: 'pt-mysql-summary',
  description: 'Executes pt-mysql-summary command.',
  sudo: 'never',
};

/**
 * Executors as `GET /api/extensions/hosts/` reports them: measured unable to elevate,
 * measured able, and never observed.
 */
const HOSTS = [
  {
    id: 'pmm-server',
    name: 'pmm-server',
    address: '127.0.0.1',
    can_elevate: false,
  },
  {
    id: 'extensions-mysql',
    name: 'extensions-mysql',
    address: '172.28.9.40',
    can_elevate: true,
  },
  { id: 'unprobed', name: 'unprobed', address: '10.0.0.9', can_elevate: null },
];

const EXECUTOR_HOST_FIELD = {
  type: 'host',
  name: 'executor_host',
  label: 'Execution Host',
  required: true,
};

const SUDO_FIELD = {
  type: 'bool',
  name: 'sudo',
  label: 'Run with sudo',
  required: false,
  default: false,
};

/** The merged schema the backend builds: a sudo toggle only for an optional-sudo snippet. */
function schemaFor(snippets: AtwSnippetSummary[]) {
  const offersSudo = snippets.some((snippet) => snippet.sudo === 'optional');
  return {
    shared: offersSudo
      ? [EXECUTOR_HOST_FIELD, SUDO_FIELD]
      : [EXECUTOR_HOST_FIELD],
    per_snippet: [],
  };
}

function mockApis(snippets: AtwSnippetSummary[]) {
  mockedApi.get.mockImplementation((url: string) => {
    if (url === '/extensions/hosts/') {
      return Promise.resolve({ data: HOSTS });
    }
    if (url.includes('/execution-schema/')) {
      return Promise.resolve({ data: schemaFor(snippets) });
    }
    return Promise.resolve({ data: [] });
  });
}

/** Open the pane with `snippets` already selected, as "Run again" does. */
function renderWithSelection(snippets: AtwSnippetSummary[]) {
  mockApis(snippets);
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  const ui: ReactNode = (
    <CollectPane
      incidentId="inc-1"
      rerunRequest={{
        nonce: 1,
        snippetFilename: snippets[0].name,
        remembered: { snippets, values: {} },
      }}
    />
  );
  return render(
    <QueryClientProvider client={queryClient}>
      <SnackbarProvider>{ui}</SnackbarProvider>
    </QueryClientProvider>
  );
}

async function chooseHost(name: string) {
  const user = userEvent.setup();
  await user.click(await screen.findByLabelText('Execution Host'));
  await user.click(await screen.findByRole('option', { name }));
}

const CANNOT_ELEVATE = /cannot run commands with sudo/i;

describe('CollectPane executor elevation warning', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('warns when the chosen host cannot elevate for a snippet that always runs with sudo', async () => {
    renderWithSelection([PT_STALK]);

    await chooseHost('pmm-server');

    const warning = await screen.findByText(CANNOT_ELEVATE);
    expect(warning).toHaveTextContent('pmm-server');
    expect(warning).toHaveTextContent('pt-stalk');
    // A warning, not a gate: the batch can still be submitted.
    expect(screen.getByRole('button', { name: 'Execute batch' })).toBeEnabled();
  });

  it('stays silent on a host that can elevate', async () => {
    renderWithSelection([PT_STALK]);

    await chooseHost('extensions-mysql');

    expect(
      await screen.findByDisplayValue('extensions-mysql')
    ).toBeInTheDocument();
    expect(screen.queryByText(CANNOT_ELEVATE)).not.toBeInTheDocument();
  });

  it('stays silent on a host whose capability was never observed', async () => {
    renderWithSelection([PT_STALK]);

    await chooseHost('unprobed');

    expect(await screen.findByDisplayValue('unprobed')).toBeInTheDocument();
    expect(screen.queryByText(CANNOT_ELEVATE)).not.toBeInTheDocument();
  });

  it('stays silent for a snippet that never runs with sudo', async () => {
    renderWithSelection([PT_MYSQL_SUMMARY]);

    await chooseHost('pmm-server');

    expect(await screen.findByDisplayValue('pmm-server')).toBeInTheDocument();
    expect(screen.queryByText(CANNOT_ELEVATE)).not.toBeInTheDocument();
  });

  it('warns for an optional-sudo snippet only once Run with sudo is on', async () => {
    renderWithSelection([DISK_USAGE]);

    await chooseHost('pmm-server');
    expect(await screen.findByDisplayValue('pmm-server')).toBeInTheDocument();
    expect(screen.queryByText(CANNOT_ELEVATE)).not.toBeInTheDocument();

    await userEvent.setup().click(screen.getByLabelText('Run with sudo'));

    const warning = await screen.findByText(CANNOT_ELEVATE);
    expect(warning).toHaveTextContent('Disk usage');
    expect(warning).toHaveTextContent(/turn sudo off/i);
  });

  it('names only the snippets that would run with sudo', async () => {
    renderWithSelection([PT_STALK, PT_MYSQL_SUMMARY]);

    await chooseHost('pmm-server');

    const warning = await screen.findByText(CANNOT_ELEVATE);
    expect(warning).toHaveTextContent('pt-stalk');
    expect(warning).not.toHaveTextContent('pt-mysql-summary');
  });

  it('clears the warning when a capable host is chosen instead', async () => {
    renderWithSelection([PT_STALK]);

    await chooseHost('pmm-server');
    await screen.findByText(CANNOT_ELEVATE);

    await chooseHost('extensions-mysql');

    await waitFor(() => {
      expect(screen.queryByText(CANNOT_ELEVATE)).not.toBeInTheDocument();
    });
  });
});
