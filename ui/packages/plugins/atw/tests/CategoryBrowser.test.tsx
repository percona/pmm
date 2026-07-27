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
import { CategoryBrowser } from '../src/CategoryBrowser';

vi.mock('@sep/api', () => ({
  apiClient: { get: vi.fn() },
}));

import { apiClient } from '@sep/api';
const mockedApi = apiClient as unknown as { get: ReturnType<typeof vi.fn> };

const singleRoot = [
  {
    category_root: 'MySQL',
    parent_category: 'PERFORMANCE_ISSUES',
    parent_category_label: 'Performance Issues',
    category: 'OVERALL_SLOWNESS',
    category_label: 'Overall Slowness',
    snippet_count: 1,
    snippets: [{ name: 'diag/slow-query.sh', title: 'Slow Query', description: '' }],
  },
];

function renderBrowser(ui: ReactNode) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Infinity } },
  });
  return render(<QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>);
}

describe('CategoryBrowser', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('hides the Category control when the listing has a single root', async () => {
    mockedApi.get.mockResolvedValue({ data: singleRoot });

    renderBrowser(<CategoryBrowser onSnippetsChange={vi.fn()} />);

    await waitFor(() => {
      expect(screen.getByRole('combobox', { name: 'Subcategory 1' })).toBeTruthy();
    });
    expect(screen.queryByRole('combobox', { name: /^Category$/ })).toBeNull();
    expect(screen.getByRole('combobox', { name: 'Subcategory 2' })).toBeTruthy();
  });

  it('shows the Category control when the listing has multiple roots', async () => {
    mockedApi.get.mockResolvedValue({
      data: [
        ...singleRoot,
        {
          category_root: 'PostgreSQL',
          parent_category: 'PERFORMANCE_ISSUES',
          parent_category_label: 'Performance Issues',
          category: 'OVERALL_SLOWNESS',
          category_label: 'Overall Slowness',
          snippet_count: 1,
          snippets: [{ name: 'diag/other.sh', title: 'Other', description: '' }],
        },
      ],
    });

    renderBrowser(<CategoryBrowser onSnippetsChange={vi.fn()} />);

    await waitFor(() => {
      expect(screen.getByRole('combobox', { name: /^Category$/ })).toBeTruthy();
    });
    expect(screen.getByRole('combobox', { name: 'Subcategory 1' })).toBeTruthy();
  });

  it('surfaces a load error', async () => {
    mockedApi.get.mockRejectedValue(new Error('boom'));

    renderBrowser(<CategoryBrowser onSnippetsChange={vi.fn()} />);

    await waitFor(() => {
      expect(screen.getByText(/Failed to load ATW categories/i)).toBeTruthy();
    });
  });
});
