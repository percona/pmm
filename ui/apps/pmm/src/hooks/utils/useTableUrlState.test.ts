import { renderHook } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { usePerconaTableUrlState } from '@percona/percona-ui';
import reactRouter from 'react-router-dom';
import { cloneColumnFilters, useTableUrlState } from './useTableUrlState';

vi.mock('@percona/percona-ui', () => ({
  usePerconaTableUrlState: vi.fn(),
}));

const setup = (params: string) => {
  const searchParams = new URLSearchParams(params);
  const setSearchParams = vi.fn();

  vi.spyOn(reactRouter, 'useSearchParams').mockReturnValue([searchParams, setSearchParams]);
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

describe('cloneColumnFilters', () => {
  it('clones range filter value tuples', () => {
    const filters = [{ id: 'queryExecutionDurationMs', value: ['1', ''] }];
    const cloned = cloneColumnFilters(filters);

    expect(cloned).toEqual(filters);
    expect(cloned[0].value).not.toBe(filters[0].value);
  });
});

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

  it('passes cloned range filter values to the table', () => {
    const rangeFilters = [{ id: 'queryExecutionDurationMs', value: ['2', ''] }];
    setup('');

    vi.mocked(usePerconaTableUrlState).mockReturnValue({
      tableState: {
        state: {
          columnFilters: rangeFilters,
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
          columnFilters: rangeFilters,
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

    const { result } = renderHook(() => useTableUrlState({ paramPrefix: 'overview' }));

    expect(result.current.tableProps.state.columnFilters).toEqual(rangeFilters);
    expect(result.current.tableProps.state.columnFilters[0].value).not.toBe(
      rangeFilters[0].value
    );
  });

  it('preserves additionalState keys such as rowSelection in tableProps.state', () => {
    const rowSelection = { 'session-1': true };
    setup('');

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
          rowSelection,
        },
        onColumnFiltersChange: vi.fn(),
        onGlobalFilterChange: vi.fn(),
        onSortingChange: vi.fn(),
        onPaginationChange: vi.fn(),
        onShowColumnFiltersChange: vi.fn(),
        onShowGlobalFilterChange: vi.fn(),
      },
    });

    const { result } = renderHook(() => useTableUrlState({ paramPrefix: 'sessions' }));

    expect(result.current.tableProps.state.rowSelection).toEqual(rowSelection);
  });

  it('updates rowSelection in tableProps.state when additionalState changes', () => {
    setup('');

    const baseReturn = {
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
          rowSelection: {} as Record<string, boolean>,
        },
        onColumnFiltersChange: vi.fn(),
        onGlobalFilterChange: vi.fn(),
        onSortingChange: vi.fn(),
        onPaginationChange: vi.fn(),
        onShowColumnFiltersChange: vi.fn(),
        onShowGlobalFilterChange: vi.fn(),
      },
    };

    vi.mocked(usePerconaTableUrlState).mockReturnValue(baseReturn);

    const { result, rerender } = renderHook(() => useTableUrlState({ paramPrefix: 'sessions' }));

    expect(result.current.tableProps.state.rowSelection).toEqual({});

    vi.mocked(usePerconaTableUrlState).mockReturnValue({
      ...baseReturn,
      tableProps: {
        ...baseReturn.tableProps,
        state: {
          ...baseReturn.tableProps.state,
          rowSelection: { 'session-1': true },
        },
      },
    });
    rerender();

    expect(result.current.tableProps.state.rowSelection).toEqual({ 'session-1': true });
  });

  it('keeps the same columnFilters reference when filter values are unchanged', () => {
    const rangeFilters = [{ id: 'queryExecutionDurationMs', value: ['2', ''] }];
    setup('');

    vi.mocked(usePerconaTableUrlState).mockReturnValue({
      tableState: {
        state: {
          columnFilters: rangeFilters,
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
          columnFilters: rangeFilters,
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

    const { result, rerender } = renderHook(() => useTableUrlState({ paramPrefix: 'overview' }));
    const firstReference = result.current.tableProps.state.columnFilters;

    rerender();

    expect(result.current.tableProps.state.columnFilters).toBe(firstReference);
  });
});
