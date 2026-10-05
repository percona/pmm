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
import type { ReactNode } from 'react';
import { CollectPane } from '../src/CollectPane';
import type { AtwRerunRequest } from '../src/types';

let mockCanMutate = true;

vi.mock('@pmm-extensions/api', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@pmm-extensions/api')>()),
  apiClient: { get: vi.fn(), post: vi.fn() },
  useAuth: () => ({ isAdmin: mockCanMutate, canMutate: mockCanMutate }),
}));

beforeEach(() => {
  mockCanMutate = true;
});

import { apiClient } from '@pmm-extensions/api';
const mockedApi = apiClient as unknown as { get: ReturnType<typeof vi.fn> };

const SNIPPET = {
  name: 'diag/slow-query.sh',
  title: 'Slow Query Diagnostics',
  description: 'Runs a slow-query capture.',
};

/** A one-field merged schema, so a remembered value has somewhere to land. */
const SCHEMA = {
  shared: [{ type: 'string', name: 'note', label: 'Operator note' }],
  per_snippet: [],
};

function mockApis() {
  mockedApi.get.mockImplementation((url: string) => {
    if (url.startsWith('/apps/atw/snippets/')) {
      return Promise.resolve({
        data: { items: [SNIPPET], total: 1, offset: 0, limit: 50 },
      });
    }
    if (url.includes('/execution-schema/')) {
      return Promise.resolve({ data: SCHEMA });
    }
    return Promise.resolve({ data: [] });
  });
}

function renderPane(ui: ReactNode, queryClient = newQueryClient()) {
  const result = render(
    <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>
  );
  return {
    ...result,
    /** Re-renders with the same client, so a cached query need not refetch. */
    rerenderWithSameClient: (next: ReactNode) =>
      result.rerender(
        <QueryClientProvider client={queryClient}>{next}</QueryClientProvider>
      ),
  };
}

function newQueryClient() {
  return new QueryClient({ defaultOptions: { queries: { retry: false } } });
}

describe('CollectPane rerun requests', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockApis();
  });

  it('prefills the form from a remembered dispatch', async () => {
    const rerunRequest: AtwRerunRequest = {
      nonce: 1,
      snippetFilename: SNIPPET.name,
      remembered: { snippets: [SNIPPET], values: { note: 'seeded value' } },
    };

    renderPane(<CollectPane incidentId="inc-1" rerunRequest={rerunRequest} />);

    await waitFor(() => {
      expect(screen.getByLabelText('Operator note')).toHaveValue(
        'seeded value'
      );
    });
  });

  it('re-seeds the form when the same snippet is requested again with a new nonce', async () => {
    const first: AtwRerunRequest = {
      nonce: 1,
      snippetFilename: SNIPPET.name,
      remembered: { snippets: [SNIPPET], values: { note: 'first run' } },
    };

    const { rerenderWithSameClient } = renderPane(
      <CollectPane incidentId="inc-1" rerunRequest={first} />
    );
    await waitFor(() => {
      expect(screen.getByLabelText('Operator note')).toHaveValue('first run');
    });

    const second: AtwRerunRequest = {
      nonce: 2,
      snippetFilename: SNIPPET.name,
      remembered: { snippets: [SNIPPET], values: { note: 'second run' } },
    };
    rerenderWithSameClient(
      <CollectPane incidentId="inc-1" rerunRequest={second} />
    );

    await waitFor(
      () => {
        expect(screen.getByLabelText('Operator note')).toHaveValue(
          'second run'
        );
      },
      { timeout: 3000 }
    );
  });

  it('reselects the snippet by name when no dispatch is remembered', async () => {
    const rerunRequest: AtwRerunRequest = {
      nonce: 1,
      snippetFilename: SNIPPET.name,
    };

    renderPane(<CollectPane incidentId="inc-1" rerunRequest={rerunRequest} />);

    // Resolved through the exact-name search, since only the filename is
    // known — the snippet's title then shows as the picker's selected chip.
    await waitFor(
      () => {
        expect(screen.getByText(SNIPPET.title)).toBeInTheDocument();
      },
      { timeout: 3000 }
    );
    // No remembered payload, so the field falls back to the schema's own default.
    await waitFor(() => {
      expect(screen.getByLabelText('Operator note')).toHaveValue('');
    });
  });

  it('reports when the snippet cannot be found by name', async () => {
    mockedApi.get.mockImplementation((url: string) => {
      if (url.startsWith('/apps/atw/snippets/')) {
        return Promise.resolve({
          data: { items: [], total: 0, offset: 0, limit: 50 },
        });
      }
      return Promise.resolve({ data: [] });
    });
    const rerunRequest: AtwRerunRequest = {
      nonce: 1,
      snippetFilename: 'diag/renamed-away.sh',
    };

    renderPane(<CollectPane incidentId="inc-1" rerunRequest={rerunRequest} />);

    await waitFor(
      () => {
        expect(
          screen.getByText(/Could not find.*diag\/renamed-away\.sh/)
        ).toBeInTheDocument();
      },
      { timeout: 3000 }
    );
  });
});
