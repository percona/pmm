import { useCallback, useMemo } from 'react';
import { useSearchParams } from 'react-router-dom';
import {
  stableDependencyKey,
  usePerconaTableUrlState,
  type MRT_ColumnFiltersState,
  type MRT_Updater,
  type UsePerconaTableUrlStateOptions,
  type UsePerconaTableUrlStateResult,
} from '@percona/percona-ui';

export type UseTableUrlStateOptions = Omit<
  UsePerconaTableUrlStateOptions,
  'searchParams' | 'setSearchParams'
>;

export type UseTableUrlStateResult = UsePerconaTableUrlStateResult;

// MRT range filters mutate filter value arrays in place. With controlled URL
// state that can mutate React state before setState runs, so isSameTableState
// sees no change and the table skips re-filtering until another filter updates.
export const cloneColumnFilters = (
  filters: MRT_ColumnFiltersState
): MRT_ColumnFiltersState =>
  filters.map(({ id, value }) => ({
    id,
    value: Array.isArray(value) ? [...value] : value,
  }));

export const useTableUrlState = (
  options: UseTableUrlStateOptions = {}
): UseTableUrlStateResult => {
  const [searchParams, setSearchParams] = useSearchParams();

  const { tableState, tableProps } = usePerconaTableUrlState({
    searchParams,
    setSearchParams,
    ...options,
  });

  const {
    state: {
      columnFilters,
      globalFilter,
      sorting,
      pagination,
      showColumnFilters,
      showGlobalFilter,
      ...additionalTableState
    },
    onGlobalFilterChange,
    onSortingChange,
    onPaginationChange,
    onShowColumnFiltersChange,
    onShowGlobalFilterChange,
    onColumnFiltersChange: onPerconaColumnFiltersChange,
  } = tableProps;

  const columnFiltersKey = stableDependencyKey(columnFilters);

  // Keep a stable clone reference while filter values are unchanged. MRT
  // re-syncs range inputs from column.getFilterValue() whenever that array
  // reference changes, which causes visible flicker during typing/refetches.
  const columnFiltersForTable = useMemo(
    () => cloneColumnFilters(columnFilters),
    // columnFiltersKey tracks columnFilters by value so the clone reference stays
    // stable across parent re-renders (e.g. query refetches) with unchanged filters.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [columnFiltersKey]
  );

  const onColumnFiltersChange = useCallback(
    (updater: MRT_Updater<MRT_ColumnFiltersState>) => {
      onPerconaColumnFiltersChange((prev) => {
        const resolved =
          updater instanceof Function
            ? updater(cloneColumnFilters(prev))
            : updater;

        return cloneColumnFilters(resolved);
      });
    },
    [onPerconaColumnFiltersChange]
  );

  const tablePropsWithImmutableFilters = useMemo(
    () => ({
      state: {
        columnFilters: columnFiltersForTable,
        globalFilter,
        sorting,
        pagination,
        showColumnFilters,
        showGlobalFilter,
        ...additionalTableState,
      },
      onColumnFiltersChange,
      onGlobalFilterChange,
      onSortingChange,
      onPaginationChange,
      onShowColumnFiltersChange,
      onShowGlobalFilterChange,
    }),
    [
      columnFiltersForTable,
      globalFilter,
      sorting,
      pagination,
      showColumnFilters,
      showGlobalFilter,
      additionalTableState,
      onColumnFiltersChange,
      onGlobalFilterChange,
      onSortingChange,
      onPaginationChange,
      onShowColumnFiltersChange,
      onShowGlobalFilterChange,
    ]
  );

  return {
    tableState,
    tableProps: tablePropsWithImmutableFilters,
  };
};
