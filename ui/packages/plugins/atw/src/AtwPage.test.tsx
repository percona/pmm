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

import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  useSnippetPluginExecution,
  useSnippetPluginSchema,
} from '@sep/framework';
import { AtwPage } from './AtwPage';
import { useAtwCategories } from './hooks';

const mocks = vi.hoisted(() => ({
  useAtwCategories: vi.fn(),
  useSnippetPluginSchema: vi.fn(),
  useSnippetPluginExecution: vi.fn(),
}));

vi.mock('@sep/framework', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@sep/framework')>();
  return {
    ...actual,
    SchemaFormRenderer: () => <div>Schema form</div>,
    buildSnippetExecutionFormPayload: (values: Record<string, unknown>) =>
      values,
    useSnippetPluginSchema: mocks.useSnippetPluginSchema,
    useSnippetPluginExecution: mocks.useSnippetPluginExecution,
  };
});

vi.mock('notistack', () => ({
  useSnackbar: () => ({ enqueueSnackbar: vi.fn() }),
}));

vi.mock('./hooks', () => ({
  useAtwCategories: mocks.useAtwCategories,
}));

const mockUseAtwCategories = vi.mocked(useAtwCategories);
const mockUseSnippetPluginSchema = vi.mocked(useSnippetPluginSchema);
const mockUseSnippetPluginExecution = vi.mocked(useSnippetPluginExecution);

describe('AtwPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockUseAtwCategories.mockReturnValue({
      data: [
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
              description: 'Collects slow-query and processlist data.',
            },
          ],
        },
      ],
      isLoading: false,
      error: null,
    } as unknown as ReturnType<typeof useAtwCategories>);
    mockUseSnippetPluginSchema.mockReturnValue({
      data: null,
      isLoading: false,
      error: null,
    } as unknown as ReturnType<typeof useSnippetPluginSchema>);
    mockUseSnippetPluginExecution.mockReturnValue({
      mutate: vi.fn(),
      isPending: false,
      isError: false,
      error: null,
    } as unknown as ReturnType<typeof useSnippetPluginExecution>);
  });

  it('hides the Category control when the listing has a single root', () => {
    render(<AtwPage />);

    expect(screen.queryByRole('combobox', { name: /^Category$/ })).toBeNull();
    expect(
      screen.getByRole('combobox', { name: 'Subcategory 1' })
    ).toBeTruthy();
    expect(
      screen.getByRole('combobox', { name: 'Subcategory 2' })
    ).toBeTruthy();
    expect(screen.getByRole('combobox', { name: 'Snippet' })).toBeTruthy();
  });

  it('shows the Category control when the listing has multiple roots', () => {
    mockUseAtwCategories.mockReturnValue({
      data: [
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
              description: 'Collects slow-query and processlist data.',
            },
          ],
        },
        {
          category_root: 'PostgreSQL',
          parent_category: 'PERFORMANCE_ISSUES',
          parent_category_label: 'Performance Issues',
          category: 'OVERALL_SLOWNESS',
          category_label: 'Overall Slowness',
          snippet_count: 1,
          snippets: [
            {
              name: 'diag/other.sh',
              title: 'Other',
              description: 'Second root fixture.',
            },
          ],
        },
      ],
      isLoading: false,
      error: null,
    } as unknown as ReturnType<typeof useAtwCategories>);

    render(<AtwPage />);

    expect(screen.getByRole('combobox', { name: /^Category$/ })).toBeTruthy();
    expect(
      screen.getByRole('combobox', { name: 'Subcategory 1' })
    ).toBeTruthy();
  });
});
