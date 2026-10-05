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
import type { ReactNode } from 'react';
import {
  CategoryFilters,
  parentFilterOptions,
  rootFilterOptions,
  snippetsForFilters,
} from '../src/CategoryFilters';
import type { AtwCategoryListing } from '../src/types';

vi.mock('@pmm-extensions/api', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@pmm-extensions/api')>()),
  apiClient: { get: vi.fn() },
}));

import { apiClient } from '@pmm-extensions/api';
const mockedApi = apiClient as unknown as { get: ReturnType<typeof vi.fn> };

const mysqlLeaf: AtwCategoryListing = {
  category_root: 'MySQL',
  parent_category: 'PERFORMANCE_ISSUES',
  parent_category_label: 'Performance Issues',
  category: 'OVERALL_SLOWNESS',
  category_label: 'Overall Slowness',
  snippet_count: 1,
  snippets: [
    { name: 'diag/slow-query.sh', title: 'Slow Query', description: '' },
  ],
};

const mysqlLocksLeaf: AtwCategoryListing = {
  category_root: 'MySQL',
  parent_category: 'LOCKS',
  parent_category_label: 'Locks',
  category: 'DEADLOCKS',
  category_label: 'Deadlocks',
  snippet_count: 2,
  snippets: [
    { name: 'diag/deadlock.sh', title: 'Deadlock', description: '' },
    { name: 'diag/lock-wait.sh', title: 'Lock Wait', description: '' },
  ],
};

const postgresLeaf: AtwCategoryListing = {
  category_root: 'PostgreSQL',
  parent_category: 'PERFORMANCE_ISSUES',
  parent_category_label: 'Performance Issues',
  category: 'OVERALL_SLOWNESS',
  category_label: 'Overall Slowness',
  snippet_count: 1,
  snippets: [{ name: 'diag/other.sh', title: 'Other', description: '' }],
};

function renderFilters(ui: ReactNode) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Infinity } },
  });
  return {
    queryClient,
    ...render(
      <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>
    ),
  };
}

describe('rootFilterOptions / parentFilterOptions / snippetsForFilters', () => {
  const listing = [mysqlLeaf, mysqlLocksLeaf, postgresLeaf];

  it('counts unique snippet filenames per root from the listing only', () => {
    expect(rootFilterOptions(listing)).toEqual([
      { root: 'MySQL', count: 3 },
      { root: 'PostgreSQL', count: 1 },
    ]);
  });

  it('does not invent empty roots absent from the response', () => {
    expect(
      rootFilterOptions(listing).map((option) => option.root)
    ).not.toContain('ProxySQL');
  });

  it('counts unique snippet filenames per problem area under a root', () => {
    expect(parentFilterOptions(listing, 'MySQL')).toEqual([
      {
        value: 'PERFORMANCE_ISSUES',
        label: 'Performance Issues',
        count: 1,
      },
      { value: 'LOCKS', label: 'Locks', count: 2 },
    ]);
  });

  it('does not double-count a script that appears on several leaves', () => {
    const crashRestartOk: AtwCategoryListing = {
      category_root: 'MySQL',
      parent_category: 'CRASHES',
      parent_category_label: 'Crashes',
      category: 'SERVER_CRASHED_RESTART_SUCCESSFUL',
      category_label: 'Server crashed - Restart Successful',
      snippet_count: 2,
      snippets: [
        { name: 'pt-mysql-summary.sh', title: 'pt-mysql-summary', description: '' },
        {
          name: 'mysql_log_extractor.sh',
          title: 'MySQL Log Extractor',
          description: '',
        },
      ],
    };
    const crashRestartFail: AtwCategoryListing = {
      category_root: 'MySQL',
      parent_category: 'CRASHES',
      parent_category_label: 'Crashes',
      category: 'SERVER_CRASHED_RESTART_NOT_SUCCESSFUL',
      category_label: 'Server crashed - Restart Not Successful',
      snippet_count: 2,
      snippets: [
        {
          name: 'mysql_config_files.sh',
          title: 'MySQL Config File Discovery Script',
          description: '',
        },
        {
          name: 'mysql_log_extractor.sh',
          title: 'MySQL Log Extractor',
          description: '',
        },
      ],
    };
    const overlapping = [crashRestartOk, crashRestartFail];

    // Summing snippet_count would be 4; the picker shows 3 unique scripts.
    expect(parentFilterOptions(overlapping, 'MySQL')).toEqual([
      { value: 'CRASHES', label: 'Crashes', count: 3 },
    ]);
    expect(rootFilterOptions(overlapping)).toEqual([
      { root: 'MySQL', count: 3 },
    ]);
    expect(
      snippetsForFilters(overlapping, 'MySQL', 'CRASHES').map((s) => s.name)
    ).toEqual([
      'pt-mysql-summary.sh',
      'mysql_log_extractor.sh',
      'mysql_config_files.sh',
    ]);
  });

  it('returns no snippets until a root is selected', () => {
    expect(snippetsForFilters(listing, '', '')).toEqual([]);
  });

  it('unions every leaf under a root when no problem area is selected', () => {
    expect(
      snippetsForFilters(listing, 'MySQL', '').map((snippet) => snippet.name)
    ).toEqual([
      'diag/slow-query.sh',
      'diag/deadlock.sh',
      'diag/lock-wait.sh',
    ]);
  });

  it('narrows to one problem area when a parent is selected', () => {
    expect(
      snippetsForFilters(listing, 'MySQL', 'LOCKS').map(
        (snippet) => snippet.name
      )
    ).toEqual(['diag/deadlock.sh', 'diag/lock-wait.sh']);
  });
});

describe('CategoryFilters', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders database chips with counts and no cascading dropdowns', async () => {
    mockedApi.get.mockResolvedValue({
      data: [mysqlLeaf, mysqlLocksLeaf, postgresLeaf],
    });

    renderFilters(<CategoryFilters onSnippetsChange={vi.fn()} />);

    expect(
      await screen.findByRole('button', { name: 'MySQL (3)' })
    ).toBeTruthy();
    expect(
      screen.getByRole('button', { name: 'PostgreSQL (1)' })
    ).toBeTruthy();
    expect(screen.queryByRole('combobox')).toBeNull();
    expect(screen.queryByText('ProxySQL')).toBeNull();
    // Problem-area chips wait for a root selection.
    expect(
      screen.queryByRole('button', { name: /Performance Issues/ })
    ).toBeNull();
  });

  it('reports the root union, then narrows on problem-area selection', async () => {
    mockedApi.get.mockResolvedValue({
      data: [mysqlLeaf, mysqlLocksLeaf, postgresLeaf],
    });
    const onSnippetsChange = vi.fn();

    renderFilters(<CategoryFilters onSnippetsChange={onSnippetsChange} />);

    await userEvent.click(
      await screen.findByRole('button', { name: 'MySQL (3)' })
    );

    await waitFor(() => {
      expect(onSnippetsChange).toHaveBeenLastCalledWith([
        mysqlLeaf.snippets[0],
        mysqlLocksLeaf.snippets[0],
        mysqlLocksLeaf.snippets[1],
      ]);
    });

    expect(
      screen.getByRole('button', { name: 'Performance Issues (1)' })
    ).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Locks (2)' })).toBeTruthy();

    await userEvent.click(screen.getByRole('button', { name: 'Locks (2)' }));

    await waitFor(() => {
      expect(onSnippetsChange).toHaveBeenLastCalledWith([
        mysqlLocksLeaf.snippets[0],
        mysqlLocksLeaf.snippets[1],
      ]);
    });
  });

  it('clears filters and reports an empty snippet list', async () => {
    mockedApi.get.mockResolvedValue({ data: [mysqlLeaf] });
    const onSnippetsChange = vi.fn();

    renderFilters(<CategoryFilters onSnippetsChange={onSnippetsChange} />);

    await userEvent.click(
      await screen.findByRole('button', { name: 'MySQL (1)' })
    );
    await waitFor(() => {
      expect(onSnippetsChange).toHaveBeenLastCalledWith(mysqlLeaf.snippets);
    });

    await userEvent.click(screen.getByRole('button', { name: 'Clear filters' }));

    await waitFor(() => {
      expect(onSnippetsChange).toHaveBeenLastCalledWith([]);
    });
    expect(
      screen.queryByRole('button', { name: /Performance Issues/ })
    ).toBeNull();
  });

  it('clears selection when the selected root disappears from the listing', async () => {
    mockedApi.get.mockResolvedValue({
      data: [mysqlLeaf, postgresLeaf],
    });
    const onSnippetsChange = vi.fn();
    const { queryClient } = renderFilters(
      <CategoryFilters onSnippetsChange={onSnippetsChange} />
    );

    await userEvent.click(
      await screen.findByRole('button', { name: 'MySQL (1)' })
    );
    await waitFor(() => {
      expect(onSnippetsChange).toHaveBeenLastCalledWith(mysqlLeaf.snippets);
    });

    queryClient.setQueryData(['atw', 'categories'], [postgresLeaf]);

    await waitFor(() => {
      expect(onSnippetsChange).toHaveBeenLastCalledWith([]);
    });
    expect(screen.queryByRole('button', { name: 'Clear filters' })).toBeNull();
    expect(screen.queryByRole('button', { name: /MySQL/ })).toBeNull();
    expect(
      screen.getByRole('button', { name: 'PostgreSQL (1)' })
    ).toBeTruthy();
  });

  it('surfaces a load error', async () => {
    mockedApi.get.mockRejectedValue(new Error('boom'));

    renderFilters(<CategoryFilters onSnippetsChange={vi.fn()} />);

    await waitFor(() => {
      expect(screen.getByText(/Failed to load ATW categories/i)).toBeTruthy();
    });
  });
});
