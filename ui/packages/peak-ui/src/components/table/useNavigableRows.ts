import {
  MRT_ColumnFiltersState,
  MRT_SortingState,
  MRT_TableInstance,
  MRT_Updater,
} from 'material-react-table';
import { useCallback, useEffect, useRef, useState, type MutableRefObject } from 'react';

export type NavigableRowsScope = 'allFiltered' | 'currentPage';

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export interface UseNavigableRowsOptions<T extends Record<string, any>> {
  data: T[];
  scope?: NavigableRowsScope;
  onChange?: (rows: T[]) => void;
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export interface UseNavigableRowsTableProps<T extends Record<string, any>> {
  tableInstanceRef: MutableRefObject<MRT_TableInstance<T> | null>;
  state: {
    columnFilters: MRT_ColumnFiltersState;
    globalFilter: string;
    sorting: MRT_SortingState;
  };
  onColumnFiltersChange: (updater: MRT_Updater<MRT_ColumnFiltersState>) => void;
  onGlobalFilterChange: (updater: MRT_Updater<string>) => void;
  onSortingChange: (updater: MRT_Updater<MRT_SortingState>) => void;
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export interface UseNavigableRowsResult<T extends Record<string, any>> {
  navigableRows: T[];
  tableProps: UseNavigableRowsTableProps<T>;
  refresh: () => void;
}

const resolveUpdater = <S>(updater: MRT_Updater<S>, prev: S): S =>
  updater instanceof Function ? updater(prev) : updater;

const sameRows = <T>(a: T[], b: T[]): boolean =>
  a.length === b.length && a.every((row, index) => row === b[index]);

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function useNavigableRows<T extends Record<string, any>>({
  data,
  scope = 'allFiltered',
  onChange,
}: UseNavigableRowsOptions<T>): UseNavigableRowsResult<T> {
  const tableInstanceRef = useRef<MRT_TableInstance<T> | null>(null);
  const [columnFilters, setColumnFilters] = useState<MRT_ColumnFiltersState>([]);
  const [globalFilter, setGlobalFilter] = useState<string>('');
  const [sorting, setSorting] = useState<MRT_SortingState>([]);
  const [navigableRows, setNavigableRows] = useState<T[]>(data);

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

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    refresh();
  }, [columnFilters, globalFilter, sorting, data, refresh]);

  useEffect(() => {
    onChangeRef.current?.(navigableRows);
  }, [navigableRows]);

  const tableProps: UseNavigableRowsTableProps<T> = {
    tableInstanceRef,
    state: { columnFilters, globalFilter, sorting },
    onColumnFiltersChange: (updater) => setColumnFilters((prev) => resolveUpdater(updater, prev)),
    onGlobalFilterChange: (updater) => setGlobalFilter((prev) => resolveUpdater(updater, prev)),
    onSortingChange: (updater) => setSorting((prev) => resolveUpdater(updater, prev)),
  };

  return { navigableRows, tableProps, refresh };
}

export default useNavigableRows;
