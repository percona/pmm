import {
  MRT_ColumnFiltersState,
  MRT_PaginationState,
  MRT_SortingState,
  MRT_TableInstance,
  MRT_Updater,
} from 'material-react-table';
import { useCallback, useEffect, useRef, useState, type MutableRefObject } from 'react';
import { DEFAULT_TABLE_STATE, type NavigableTableState } from './tableState.types';
import { resolveUpdater } from './tableState.utils';

export type NavigableRowsScope = 'allFiltered' | 'currentPage';

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export interface UseNavigableRowsOptions<T extends Record<string, any>> {
  /** All rows given to the table. */
  data: T[];
  /** Which rows count as neighbors: every row that passes the filters (default) or only the visible page. */
  scope?: NavigableRowsScope;
  /** Called whenever the ordered list of navigable rows changes. */
  onChange?: (rows: T[]) => void;
  /** Controlled table state, for example from `usePerconaTableUrlState`, when the parent owns filters and sorting. */
  tableState?: NavigableTableState;
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export interface UseNavigableRowsTableProps<T extends Record<string, any>> {
  /** Spread onto `Table` so the hook can read the rows in the order the user sees them. */
  tableInstanceRef: MutableRefObject<MRT_TableInstance<T> | null>;
  /** Filters, search text, sorting, and (for the current-page scope) pagination the hook tracks for the parent. */
  state?: {
    /** Active per-column filters. */
    columnFilters: MRT_ColumnFiltersState;
    /** Text typed in the search box. */
    globalFilter: string;
    /** Active sort order, one entry per sorted column. */
    sorting: MRT_SortingState;
    /** Current page and rows per page. */
    pagination?: MRT_PaginationState;
  };
  /** Receives the new per-column filters when the user changes them. */
  onColumnFiltersChange?: (updater: MRT_Updater<MRT_ColumnFiltersState>) => void;
  /** Receives the new search text when the user types in the search box. */
  onGlobalFilterChange?: (updater: MRT_Updater<string>) => void;
  /** Receives the new sort order when the user sorts a column. */
  onSortingChange?: (updater: MRT_Updater<MRT_SortingState>) => void;
  /** Receives the new page or rows per page when the user paginates. */
  onPaginationChange?: (updater: MRT_Updater<MRT_PaginationState>) => void;
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export interface UseNavigableRowsResult<T extends Record<string, any>> {
  /** Rows the user can step through, in the order shown: all filtered and sorted rows, or only the visible page with the current-page scope. */
  navigableRows: T[];
  /** Spread onto `Table` so the hook stays in sync with what the user sees. */
  tableProps: UseNavigableRowsTableProps<T>;
  /** Recomputes the rows on demand, for example after the data changes in place. */
  refresh: () => void;
}

const sameRows = <T>(a: T[], b: T[]): boolean =>
  a.length === b.length && a.every((row, index) => row === b[index]);

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function useNavigableRows<T extends Record<string, any>>({
  data,
  scope = 'allFiltered',
  onChange,
  tableState,
}: UseNavigableRowsOptions<T>): UseNavigableRowsResult<T> {
  const tableInstanceRef = useRef<MRT_TableInstance<T> | null>(null);
  const [internalColumnFilters, setInternalColumnFilters] = useState<MRT_ColumnFiltersState>([]);
  const [internalGlobalFilter, setInternalGlobalFilter] = useState<string>('');
  const [internalSorting, setInternalSorting] = useState<MRT_SortingState>([]);
  const [internalPagination, setInternalPagination] = useState<MRT_PaginationState>(
    DEFAULT_TABLE_STATE.pagination
  );
  const [navigableRows, setNavigableRows] = useState<T[]>(data);

  const usesInternalPagination = !tableState && scope === 'currentPage';
  const columnFilters = tableState?.state.columnFilters ?? internalColumnFilters;
  const globalFilter = tableState?.state.globalFilter ?? internalGlobalFilter;
  const sorting = tableState?.state.sorting ?? internalSorting;
  const pagination =
    tableState?.state.pagination ??
    (usesInternalPagination ? internalPagination : DEFAULT_TABLE_STATE.pagination);

  const onChangeRef = useRef(onChange);
  useEffect(() => {
    onChangeRef.current = onChange;
  }, [onChange]);

  const computeRows = useCallback((): T[] => {
    const table = tableInstanceRef.current;
    if (!table) {
      return data;
    }
    const rowModel =
      scope === 'currentPage' ? table.getRowModel() : table.getPrePaginationRowModel();
    return rowModel.rows.map((row) => row.original);
  }, [data, scope]);

  const refresh = useCallback(() => {
    const next = computeRows();
    setNavigableRows((prev) => (sameRows(prev, next) ? prev : next));
  }, [computeRows]);

  const paginationDepKey =
    scope === 'currentPage' ? `${pagination.pageIndex}:${pagination.pageSize}` : null;

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    refresh();
  }, [columnFilters, globalFilter, sorting, paginationDepKey, data, refresh, scope]);

  useEffect(() => {
    onChangeRef.current?.(navigableRows);
  }, [navigableRows]);

  const tableProps: UseNavigableRowsTableProps<T> = tableState
    ? { tableInstanceRef }
    : {
        tableInstanceRef,
        state: {
          columnFilters,
          globalFilter,
          sorting,
          ...(usesInternalPagination ? { pagination } : {}),
        },
        onColumnFiltersChange: (updater) =>
          setInternalColumnFilters((prev) => resolveUpdater(updater, prev)),
        onGlobalFilterChange: (updater) =>
          setInternalGlobalFilter((prev) => resolveUpdater(updater, prev)),
        onSortingChange: (updater) => setInternalSorting((prev) => resolveUpdater(updater, prev)),
        ...(usesInternalPagination
          ? {
              onPaginationChange: (updater) =>
                setInternalPagination((prev) => resolveUpdater(updater, prev)),
            }
          : {}),
      };

  return { navigableRows, tableProps, refresh };
}

export default useNavigableRows;
