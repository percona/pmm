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

import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import { CollectPane } from '../src/CollectPane';

/** Flipped per test to cover the read-only (non-admin) rendering. */
let mockCanMutate = true;

vi.mock('@sep/api', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@sep/api')>()),
  apiClient: { get: vi.fn(), post: vi.fn() },
  useAuth: () => ({ isAdmin: mockCanMutate, canMutate: mockCanMutate }),
}));

beforeEach(() => {
  mockCanMutate = true;
});

import { apiClient } from '@sep/api';
const mockedApi = apiClient as unknown as { get: ReturnType<typeof vi.fn> };

/** ATW category listing: one leaf category exposing one tagged snippet. */
const CATEGORY_LISTING = [
  {
    category_root: 'MySQL',
    parent_category: 'PERFORMANCE_ISSUES',
    parent_category_label: 'Performance Issues',
    category: 'OVERALL_SLOWNESS',
    category_label: 'Overall Slowness',
    snippet_count: 1,
    snippets: [
      {
        name: 'diag/slow-query.sh',
        title: 'Slow Query Diagnostics',
        description: 'Tagged.',
      },
    ],
  },
];

/** A snippet reachable only through search — it carries no `atw` tag. */
const SEARCH_ROW = {
  name: 'ops/pt-summary.sh',
  title: 'PT Summary',
  description: 'Collects a percona-toolkit system summary.',
};

interface SearchPage {
  items: (typeof SEARCH_ROW)[];
  total?: number;
}

/**
 * Route each mocked GET by path; the search page is configurable per test.
 *
 * Search and the category listing share the `/apps/atw` prefix, so the more
 * specific search path must be matched before the listing fallback below.
 */
function mockApis(page: SearchPage = { items: [SEARCH_ROW] }): void {
  mockedApi.get.mockImplementation((url: string) => {
    if (url.startsWith('/apps/atw/snippets/')) {
      return Promise.resolve({
        data: {
          items: page.items,
          total: page.total ?? page.items.length,
          offset: 0,
          limit: 50,
        },
      });
    }
    if (url.includes('/execution-schema/')) {
      return Promise.resolve({ data: { shared: [], per_snippet: [] } });
    }
    return Promise.resolve({ data: CATEGORY_LISTING });
  });
}

function renderPane(ui: ReactNode) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>
  );
}

/** Every snippet-search request issued so far, with its query params. */
function searchCalls(): { search?: string; limit?: number; offset?: number }[] {
  return mockedApi.get.mock.calls
    .filter((call) => String(call[0]).startsWith('/apps/atw/snippets/'))
    .map((call) => (call[1] as { params: Record<string, unknown> }).params);
}

/** Type the term into the picker one key at a time, inside one debounce window. */
async function typeSearch(term: string): Promise<HTMLElement> {
  const input = await screen.findByRole('combobox', { name: 'Snippets' });
  await userEvent.type(input, term);
  return input;
}

describe('CollectPane snippet search', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('debounces typing into one search request', async () => {
    mockApis();
    renderPane(<CollectPane incidentId="inc-1" />);

    await typeSearch('summary');

    await waitFor(() => expect(searchCalls()).toHaveLength(1), {
      timeout: 3000,
    });
    expect(searchCalls()[0]).toEqual({
      search: 'summary',
      offset: 0,
      limit: 50,
    });
  });

  it('searches ATW’s own route rather than the snippets app’s', async () => {
    // The snippets app is not activated on the PMM-embedded profile, so a
    // request to its route would 404 there while the category browser worked.
    mockApis();
    renderPane(<CollectPane incidentId="inc-1" />);

    await typeSearch('summary');

    await waitFor(() => expect(searchCalls()).toHaveLength(1), {
      timeout: 3000,
    });
    const urls = mockedApi.get.mock.calls.map((call) => String(call[0]));
    expect(urls).not.toEqual(
      expect.arrayContaining([expect.stringContaining('/apps/snippets')])
    );
    expect(urls.every((url) => url.startsWith('/apps/atw'))).toBe(true);
  });

  it('never sends an approval param, since the route pins approved-only', async () => {
    mockApis();
    renderPane(<CollectPane incidentId="inc-1" />);

    const input = await typeSearch('summary');
    await waitFor(() => expect(searchCalls()).toHaveLength(1), {
      timeout: 3000,
    });
    await userEvent.type(input, ' log');

    // Across every request the picker issues, not just the first: a widened set
    // would offer snippets the execute path rejects.
    await waitFor(() => expect(searchCalls().length).toBeGreaterThan(1), {
      timeout: 3000,
    });
    expect(searchCalls().every((params) => !('approval' in params))).toBe(true);
  });

  it('issues no search while the picker text is empty', async () => {
    mockApis();
    renderPane(<CollectPane incidentId="inc-1" />);

    // The category listing resolves, so the pane has settled without a search.
    await screen.findByRole('combobox', { name: 'Subcategory 1' });
    await new Promise((resolve) => setTimeout(resolve, 500));

    expect(searchCalls()).toHaveLength(0);
  });

  it('offers a snippet the category browser never exposes, and keeps it as a chip', async () => {
    mockApis();
    renderPane(<CollectPane incidentId="inc-1" />);

    const input = await typeSearch('summary');

    const option = await screen.findByRole(
      'option',
      { name: /PT Summary/ },
      { timeout: 3000 }
    );
    expect(option).toHaveTextContent('ops/pt-summary.sh');
    await userEvent.click(option);

    // Clearing the search must not take the selection with it.
    await userEvent.clear(input);
    await userEvent.keyboard('{Escape}');

    // The option list lives in a portal, so scoping to the picker itself proves
    // the match is the selected chip rather than a leftover option row.
    const picker = input.closest('.MuiAutocomplete-root') as HTMLElement;
    await waitFor(() => {
      expect(within(picker).getByText('PT Summary')).toBeInTheDocument();
    });
  });

  it('keeps a search hit whose title does not contain the typed term', async () => {
    mockApis({ items: [SEARCH_ROW] });
    renderPane(<CollectPane incidentId="inc-1" />);

    // Matches the row's description only, so the default label-only filter
    // would have dropped it after the server had already returned it.
    await typeSearch('percona-toolkit');

    expect(
      await screen.findByRole(
        'option',
        { name: /PT Summary/ },
        { timeout: 3000 }
      )
    ).toBeVisible();
  });

  it('renders two same-titled hits as distinct rows', async () => {
    mockApis({
      items: [
        {
          name: 'ops/pt-summary.sh',
          title: 'Summary',
          description: 'Toolkit summary.',
        },
        {
          name: 'diag/summary.sh',
          title: 'Summary',
          description: 'Diagnostic summary.',
        },
      ],
    });
    // Titles are not unique by contract, and searching reaches every approved
    // snippet rather than the curated ATW subset, so a collision is reachable.
    // Without `getOptionKey` MUI keys each row on the label, and the list
    // rebuilds on every keystroke — duplicate keys risk a dropped row.
    const consoleError = vi
      .spyOn(console, 'error')
      .mockImplementation(() => {});
    renderPane(<CollectPane incidentId="inc-1" />);

    await typeSearch('summary');

    const options = await screen.findAllByRole(
      'option',
      { name: /Summary/ },
      { timeout: 3000 }
    );
    expect(options.map((option) => option.textContent)).toEqual([
      expect.stringContaining('ops/pt-summary.sh'),
      expect.stringContaining('diag/summary.sh'),
    ]);
    expect(consoleError.mock.calls.flat().join(' ')).not.toContain('same key');
    consoleError.mockRestore();
  });

  it('drops the previous term’s hits while the next search is in flight', async () => {
    // The second term never resolves, so the query is still fetching and holds
    // the first term's page as placeholder data throughout the assertion.
    mockedApi.get.mockImplementation(
      (url: string, config: { params: { search: string } }) => {
        if (url.startsWith('/apps/atw/snippets/')) {
          if (config.params.search === 'percona') {
            return Promise.resolve({
              data: { items: [SEARCH_ROW], total: 1, offset: 0, limit: 50 },
            });
          }
          return new Promise(() => {});
        }
        return Promise.resolve({ data: CATEGORY_LISTING });
      }
    );
    renderPane(<CollectPane incidentId="inc-1" />);

    // Matches the description only, so the row is visible on server provenance.
    const input = await typeSearch('percona');
    await screen.findByRole(
      'option',
      { name: /PT Summary/ },
      { timeout: 3000 }
    );

    await userEvent.type(input, 'xyz');

    // Wait until the new term's request is out before asserting: only then has
    // the debounce caught up with the box, so the row can no longer be hidden by
    // the term simply not having settled yet. What is left holding it is the
    // stand-in page from the previous term, and that must not grant provenance.
    await waitFor(
      () => {
        expect(
          searchCalls().some((params) => params.search === 'perconaxyz')
        ).toBe(true);
      },
      { timeout: 3000 }
    );

    expect(
      screen.queryByRole('option', { name: /PT Summary/ })
    ).not.toBeInTheDocument();
  });

  it('reports how many matches the fetched page left out', async () => {
    mockApis({ items: [SEARCH_ROW], total: 137 });
    renderPane(<CollectPane incidentId="inc-1" />);

    await typeSearch('log');

    const notice = await screen.findByText(
      /Showing the first 1 of 137 snippets/,
      undefined,
      {
        timeout: 3000,
      }
    );
    expect(notice).toHaveTextContent(
      'Type more of the name or description to narrow the results.'
    );
  });

  it('shows no truncation notice when the page holds every match', async () => {
    mockApis({ items: [SEARCH_ROW] });
    renderPane(<CollectPane incidentId="inc-1" />);

    await typeSearch('summary');

    await screen.findByRole(
      'option',
      { name: /PT Summary/ },
      { timeout: 3000 }
    );
    expect(screen.queryByText(/Showing the first/)).not.toBeInTheDocument();
  });

  it('surfaces a failed search instead of leaving the picker silently empty', async () => {
    mockedApi.get.mockImplementation((url: string) => {
      if (url.startsWith('/apps/atw/snippets/')) {
        return Promise.reject(new Error('search backend down'));
      }
      return Promise.resolve({ data: CATEGORY_LISTING });
    });
    renderPane(<CollectPane incidentId="inc-1" />);

    await typeSearch('summary');

    const alert = await screen.findByText(/Snippet search failed/, undefined, {
      timeout: 3000,
    });
    expect(alert).toHaveTextContent('search backend down');
  });
});

describe('CollectPane — write access', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  async function selectOneSnippet() {
    mockApis();
    renderPane(<CollectPane incidentId="inc-1" />);
    await typeSearch('summary');
    const option = await screen.findByRole(
      'option',
      { name: /PT Summary/ },
      { timeout: 3000 }
    );
    await userEvent.click(option);
  }

  it('renders the batch execute form for a session that may mutate', async () => {
    await selectOneSnippet();

    expect(
      await screen.findByRole(
        'button',
        { name: /Execute batch/i },
        { timeout: 3000 }
      )
    ).toBeInTheDocument();
  });

  it('renders no batch execute form for a non-admin', async () => {
    mockCanMutate = false;
    await selectOneSnippet();

    // The picker still works — only the execute form is withheld.
    await waitFor(() => {
      expect(
        screen.queryByRole('button', { name: /Execute batch/i })
      ).not.toBeInTheDocument();
    });
  });

  it('fetches no execution schema for a non-admin', async () => {
    // The form is the only consumer of the merged schema, so a session that
    // cannot execute should not pay for the request.
    mockCanMutate = false;
    await selectOneSnippet();

    const schemaCalls = () =>
      mockedApi.get.mock.calls.filter((call) =>
        String(call[0]).includes('/execution-schema/')
      );
    await waitFor(() => expect(schemaCalls()).toHaveLength(0));
  });

  it('fetches the execution schema for a session that may mutate', async () => {
    await selectOneSnippet();

    await waitFor(() =>
      expect(
        mockedApi.get.mock.calls.filter((call) =>
          String(call[0]).includes('/execution-schema/')
        ).length
      ).toBeGreaterThan(0)
    );
  });
});
