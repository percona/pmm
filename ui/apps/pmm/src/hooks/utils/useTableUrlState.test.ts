import { renderHook } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import reactRouter from 'react-router-dom';
import { useTableUrlState } from './useTableUrlState';

const { usePerconaTableUrlState } = vi.hoisted(() => ({
  usePerconaTableUrlState: vi.fn(),
}));

vi.mock('@percona/percona-ui', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@percona/percona-ui')>();
  return {
    ...actual,
    usePerconaTableUrlState,
  };
});

const setup = (params: string) => {
  const searchParams = new URLSearchParams(params);
  const setSearchParams = vi.fn();

  vi.spyOn(reactRouter, 'useSearchParams').mockReturnValue([
    searchParams,
    setSearchParams,
  ]);
  vi.mocked(usePerconaTableUrlState).mockReturnValue({
    tableState: {
      state: {
        columnFilters: [],
        globalFilter: '',
        sorting: [],
        pagination: { pageIndex: 0, pageSize: 10 },
      },
      onColumnFiltersChange: vi.fn(),
      onGlobalFilterChange: vi.fn(),
      onSortingChange: vi.fn(),
      onPaginationChange: vi.fn(),
    },
    tableProps: {
      state: {
        columnFilters: [],
        globalFilter: '',
        sorting: [],
        pagination: { pageIndex: 0, pageSize: 10 },
        showColumnFilters: false,
        showGlobalFilter: false,
      },
      onColumnFiltersChange: vi.fn(),
      onGlobalFilterChange: vi.fn(),
      onSortingChange: vi.fn(),
      onPaginationChange: vi.fn(),
      onShowColumnFiltersChange: vi.fn(),
      onShowGlobalFilterChange: vi.fn(),
    },
  });

  return { searchParams, setSearchParams };
};

describe('useTableUrlState', () => {
  it('passes react-router search params to percona useTableUrlState', () => {
    const { searchParams, setSearchParams } = setup(
      'serviceIds=123&overview.sort=queryText:desc'
    );

    renderHook(() =>
      useTableUrlState({
        paramPrefix: 'overview',
      })
    );

    expect(usePerconaTableUrlState).toHaveBeenCalledWith({
      searchParams,
      setSearchParams,
      paramPrefix: 'overview',
    });
  });

  it('forwards optional hook configuration', () => {
    const { searchParams, setSearchParams } = setup('');

    renderHook(() =>
      useTableUrlState({
        paramPrefix: 'sessions',
        debounceMs: 500,
        replace: false,
        sync: { pagination: false },
      })
    );

    expect(usePerconaTableUrlState).toHaveBeenCalledWith({
      searchParams,
      setSearchParams,
      paramPrefix: 'sessions',
      debounceMs: 500,
      replace: false,
      sync: { pagination: false },
    });
  });
});
