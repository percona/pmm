import { act, render, renderHook, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { useRef, useState } from 'react';
import { usePerconaTableUrlState } from './usePerconaTableUrlState';

function UrlStateHarness({
  initialSearch = '',
  paramPrefix,
}: {
  initialSearch?: string;
  paramPrefix?: string;
}) {
  const [searchParams, setSearchParams] = useState(() => new URLSearchParams(initialSearch));
  const { tableProps } = usePerconaTableUrlState({
    searchParams,
    setSearchParams,
    paramPrefix,
    debounceMs: 0,
  });

  return (
    <>
      <div data-testid="global-filter">{tableProps.state.globalFilter}</div>
      <div data-testid="sorting">
        {tableProps.state.sorting.map((s) => `${s.id}:${s.desc ? 'desc' : 'asc'}`).join(',')}
      </div>
      <div data-testid="filters">
        {tableProps.state.columnFilters.map((f) => `${f.id}=${f.value}`).join(',')}
      </div>
      <div data-testid="page">{tableProps.state.pagination.pageIndex}</div>
      <div data-testid="page-size">{tableProps.state.pagination.pageSize}</div>
      <div data-testid="show-column-filters">{String(tableProps.state.showColumnFilters)}</div>
      <div data-testid="search">{searchParams.toString()}</div>
      <button
        type="button"
        data-testid="set-filter"
        onClick={() => tableProps.onColumnFiltersChange([{ id: 'group', value: 'edge' }])}
      >
        Filter
      </button>
      <button
        type="button"
        data-testid="set-sort"
        onClick={() => tableProps.onSortingChange([{ id: 'name', desc: true }])}
      >
        Sort
      </button>
      <button
        type="button"
        data-testid="set-page"
        onClick={() =>
          tableProps.onPaginationChange((prev) => ({ ...prev, pageIndex: prev.pageIndex + 1 }))
        }
      >
        Page
      </button>
      <button
        type="button"
        data-testid="set-global-filter"
        onClick={() => tableProps.onGlobalFilterChange('mysql')}
      >
        Search
      </button>
      <button
        type="button"
        data-testid="set-range-filter"
        onClick={() => tableProps.onColumnFiltersChange([{ id: 'cpu', value: ['10', ''] }])}
      >
        Range
      </button>
      <button
        type="button"
        data-testid="update-range-filter"
        onClick={() => tableProps.onColumnFiltersChange([{ id: 'cpu', value: ['25', ''] }])}
      >
        Update range
      </button>
      <button
        type="button"
        data-testid="sync-from-url"
        onClick={() => setSearchParams(new URLSearchParams('q=from-url&sort=status:asc&page=2'))}
      >
        Sync URL
      </button>
    </>
  );
}

describe('usePerconaTableUrlState', () => {
  it('initializes state from URL params', () => {
    render(
      <UrlStateHarness initialSearch="q=mysql&sort=name:desc&f.group=edge&page=2&pageSize=25" />
    );

    expect(screen.getByTestId('global-filter').textContent).toBe('mysql');
    expect(screen.getByTestId('sorting').textContent).toBe('name:desc');
    expect(screen.getByTestId('filters').textContent).toBe('group=edge');
    expect(screen.getByTestId('page').textContent).toBe('1');
    expect(screen.getByTestId('page-size').textContent).toBe('25');
    expect(screen.getByTestId('show-column-filters').textContent).toBe('true');
  });

  it('keeps column filters hidden when URL has no active filters', () => {
    render(<UrlStateHarness initialSearch="sort=name:desc" />);

    expect(screen.getByTestId('show-column-filters').textContent).toBe('false');
  });

  it('writes filter, sort, pagination, and global filter changes to the URL', async () => {
    render(<UrlStateHarness />);

    await act(async () => {
      screen.getByTestId('set-filter').click();
    });
    await waitFor(() => {
      expect(screen.getByTestId('filters').textContent).toBe('group=edge');
      expect(screen.getByTestId('search').textContent).toContain('f.group=edge');
    });

    await act(async () => {
      screen.getByTestId('set-sort').click();
    });
    await waitFor(() => {
      expect(screen.getByTestId('search').textContent).toContain('sort=name%3Adesc');
    });

    await act(async () => {
      screen.getByTestId('set-page').click();
    });
    await waitFor(() => {
      expect(screen.getByTestId('page').textContent).toBe('1');
      expect(screen.getByTestId('search').textContent).toContain('page=2');
    });

    await act(async () => {
      screen.getByTestId('set-global-filter').click();
    });
    await waitFor(() => {
      expect(screen.getByTestId('global-filter').textContent).toBe('mysql');
      expect(screen.getByTestId('search').textContent).toContain('q=mysql');
    });
  });

  it('writes range filters to the URL', async () => {
    render(<UrlStateHarness />);

    await act(async () => {
      screen.getByTestId('set-range-filter').click();
    });

    await waitFor(() => {
      expect(screen.getByTestId('filters').textContent).toBe('cpu=10,');
      expect(screen.getByTestId('search').textContent).toContain('f.cpu=%5B%2210%22%2C%22%22%5D');
    });
  });

  it('updates an existing range filter in the URL', async () => {
    render(<UrlStateHarness />);

    await act(async () => {
      screen.getByTestId('set-range-filter').click();
    });
    await waitFor(() => {
      expect(screen.getByTestId('search').textContent).toContain('f.cpu=%5B%2210%22%2C%22%22%5D');
    });

    await act(async () => {
      screen.getByTestId('update-range-filter').click();
    });
    await waitFor(() => {
      expect(screen.getByTestId('filters').textContent).toBe('cpu=25,');
      expect(screen.getByTestId('search').textContent).toContain('f.cpu=%5B%2225%22%2C%22%22%5D');
      expect(screen.getByTestId('search').textContent).not.toContain('%2210%22');
    });
  });

  it('re-reads state when searchParams change externally', async () => {
    render(<UrlStateHarness />);

    await act(async () => {
      screen.getByTestId('sync-from-url').click();
    });

    await waitFor(() => {
      expect(screen.getByTestId('global-filter').textContent).toBe('from-url');
      expect(screen.getByTestId('sorting').textContent).toBe('status:asc');
      expect(screen.getByTestId('page').textContent).toBe('1');
    });
  });

  it('still syncs external URL changes when setSearchParams ignores an internal write', async () => {
    function Harness() {
      const [searchParams, setSearchParams] = useState(() => new URLSearchParams());
      const suppressNextWriteRef = useRef(false);
      const { tableProps } = usePerconaTableUrlState({
        searchParams,
        setSearchParams: (next) => {
          if (suppressNextWriteRef.current) {
            suppressNextWriteRef.current = false;
            return;
          }
          setSearchParams(next);
        },
        debounceMs: 0,
      });

      return (
        <>
          <div data-testid="global-filter">{tableProps.state.globalFilter}</div>
          <div data-testid="search">{searchParams.toString()}</div>
          <button
            type="button"
            data-testid="set-filter"
            onClick={() => {
              suppressNextWriteRef.current = true;
              tableProps.onColumnFiltersChange([{ id: 'group', value: 'edge' }]);
            }}
          />
          <button
            type="button"
            data-testid="sync-from-url"
            onClick={() => setSearchParams(new URLSearchParams('q=from-url'))}
          />
        </>
      );
    }

    render(<Harness />);

    await act(async () => {
      screen.getByTestId('set-filter').click();
    });
    expect(screen.getByTestId('search').textContent).toBe('');

    await act(async () => {
      screen.getByTestId('sync-from-url').click();
    });
    await waitFor(() => {
      expect(screen.getByTestId('global-filter').textContent).toBe('from-url');
      expect(screen.getByTestId('search').textContent).toBe('q=from-url');
    });
  });

  it('supports paramPrefix namespacing', () => {
    render(<UrlStateHarness initialSearch="servers.q=prod&servers.page=3" paramPrefix="servers" />);

    expect(screen.getByTestId('global-filter').textContent).toBe('prod');
    expect(screen.getByTestId('page').textContent).toBe('2');
  });

  it('merges additionalState into tableProps.state', () => {
    const rowSelection = { 'srv-001': true };

    function Harness() {
      const [searchParams, setSearchParams] = useState(() => new URLSearchParams());
      const { tableProps } = usePerconaTableUrlState({
        searchParams,
        setSearchParams,
        additionalState: { rowSelection },
      });

      return <div data-testid="row-selection">{JSON.stringify(tableProps.state.rowSelection)}</div>;
    }

    render(<Harness />);

    expect(screen.getByTestId('row-selection').textContent).toBe(JSON.stringify(rowSelection));
  });

  it('keeps additionalState in tableProps.state when rowSelection updates', async () => {
    function Harness() {
      const [searchParams, setSearchParams] = useState(() => new URLSearchParams());
      const [rowSelection, setRowSelection] = useState<Record<string, boolean>>({});
      const { tableProps } = usePerconaTableUrlState({
        searchParams,
        setSearchParams,
        additionalState: { rowSelection },
      });

      return (
        <>
          <div data-testid="row-selection">{JSON.stringify(tableProps.state.rowSelection)}</div>
          <button
            type="button"
            data-testid="select-row"
            onClick={() => setRowSelection({ 'srv-001': true })}
          >
            Select
          </button>
        </>
      );
    }

    render(<Harness />);

    expect(screen.getByTestId('row-selection').textContent).toBe('{}');

    await act(async () => {
      screen.getByTestId('select-row').click();
    });

    await waitFor(() => {
      expect(screen.getByTestId('row-selection').textContent).toBe(
        JSON.stringify({ 'srv-001': true })
      );
    });
  });

  it('updates sorting locally without writing sort to the URL when sync.sort is false', async () => {
    function Harness() {
      const [searchParams, setSearchParams] = useState(() => new URLSearchParams());
      const { tableProps } = usePerconaTableUrlState({
        searchParams,
        setSearchParams,
        debounceMs: 0,
        sync: { sort: false },
      });

      return (
        <>
          <div data-testid="sorting">
            {tableProps.state.sorting.map((s) => `${s.id}:${s.desc ? 'desc' : 'asc'}`).join(',')}
          </div>
          <div data-testid="search">{searchParams.toString()}</div>
          <button
            type="button"
            data-testid="set-sort"
            onClick={() => tableProps.onSortingChange([{ id: 'name', desc: true }])}
          >
            Sort
          </button>
        </>
      );
    }

    render(<Harness />);

    await act(async () => {
      screen.getByTestId('set-sort').click();
    });

    await waitFor(() => {
      expect(screen.getByTestId('sorting').textContent).toBe('name:desc');
      expect(screen.getByTestId('search').textContent).not.toContain('sort=');
    });
  });

  it('debounced filter URL write includes sort changes made before the timeout', async () => {
    vi.useFakeTimers();

    function Harness() {
      const [searchParams, setSearchParams] = useState(() => new URLSearchParams());
      const { tableProps } = usePerconaTableUrlState({
        searchParams,
        setSearchParams,
        debounceMs: 300,
      });

      return (
        <>
          <div data-testid="search">{searchParams.toString()}</div>
          <button
            type="button"
            data-testid="set-filter"
            onClick={() => tableProps.onColumnFiltersChange([{ id: 'group', value: 'edge' }])}
          >
            Filter
          </button>
          <button
            type="button"
            data-testid="set-sort"
            onClick={() => tableProps.onSortingChange([{ id: 'name', desc: true }])}
          >
            Sort
          </button>
        </>
      );
    }

    render(<Harness />);

    await act(async () => {
      screen.getByTestId('set-filter').click();
      screen.getByTestId('set-sort').click();
      vi.advanceTimersByTime(300);
    });

    const search = screen.getByTestId('search').textContent ?? '';
    expect(search).toContain('f.group=edge');
    expect(search).toContain('sort=name%3Adesc');

    vi.useRealTimers();
  });

  it('cancels pending debounced writes when searchParams change externally', async () => {
    vi.useFakeTimers();

    function Harness() {
      const [searchParams, setSearchParams] = useState(() => new URLSearchParams());
      const { tableProps } = usePerconaTableUrlState({
        searchParams,
        setSearchParams,
        debounceMs: 300,
      });

      return (
        <>
          <div data-testid="search">{searchParams.toString()}</div>
          <button
            type="button"
            data-testid="set-filter"
            onClick={() => tableProps.onColumnFiltersChange([{ id: 'group', value: 'edge' }])}
          >
            Filter
          </button>
          <button
            type="button"
            data-testid="sync-from-url"
            onClick={() => setSearchParams(new URLSearchParams('q=from-url'))}
          >
            Sync URL
          </button>
        </>
      );
    }

    render(<Harness />);

    await act(async () => {
      screen.getByTestId('set-filter').click();
    });

    await act(async () => {
      screen.getByTestId('sync-from-url').click();
    });

    await act(async () => {
      vi.advanceTimersByTime(300);
    });

    const search = screen.getByTestId('search').textContent ?? '';
    expect(search).toBe('q=from-url');
    expect(search).not.toContain('f.group');

    vi.useRealTimers();
  });

  it('applies range filter updates when MRT mutates the previous filter array in place', async () => {
    function Harness() {
      const [searchParams, setSearchParams] = useState(() => new URLSearchParams());
      const { tableProps } = usePerconaTableUrlState({
        searchParams,
        setSearchParams,
        debounceMs: 0,
      });

      return (
        <>
          <div data-testid="filters">
            {tableProps.state.columnFilters.map((f) => `${f.id}=${f.value}`).join(',')}
          </div>
          <button
            type="button"
            data-testid="mutate-range"
            onClick={() => {
              tableProps.onColumnFiltersChange((prev) => {
                const next = prev.length
                  ? prev
                  : [{ id: 'cpu', value: ['', ''] as [string, string] }];
                (next[0].value as string[])[0] = '25';
                return next;
              });
            }}
          >
            Mutate
          </button>
        </>
      );
    }

    render(<Harness />);

    await act(async () => {
      screen.getByTestId('mutate-range').click();
    });

    await waitFor(() => {
      expect(screen.getByTestId('filters').textContent).toBe('cpu=25,');
    });
  });

  it('keeps the same columnFilters reference when filter values are unchanged', () => {
    function useHarness() {
      const [searchParams, setSearchParams] = useState(
        () => new URLSearchParams('f.cpu=%5B%222%22%2C%22%22%5D')
      );
      return usePerconaTableUrlState({ searchParams, setSearchParams });
    }

    const { result, rerender } = renderHook(() => useHarness());
    const firstReference = result.current.tableProps.state.columnFilters;

    rerender();

    expect(result.current.tableProps.state.columnFilters).toBe(firstReference);
  });

  it('does not loop when defaults and sync are inline object literals', async () => {
    const onRender = vi.fn();

    function InlineOptionsHarness() {
      onRender();
      const [searchParams, setSearchParams] = useState(() => new URLSearchParams());

      usePerconaTableUrlState({
        searchParams,
        setSearchParams,
        paramPrefix: 'overview',
        defaults: { pagination: { pageIndex: 0, pageSize: 25 } },
        sync: { globalFilter: false },
      });

      return <div data-testid="render-count">{onRender.mock.calls.length}</div>;
    }

    render(<InlineOptionsHarness />);

    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 50));
    });

    expect(onRender.mock.calls.length).toBeLessThan(5);
  });
});
