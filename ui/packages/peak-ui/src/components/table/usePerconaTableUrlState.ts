import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  MRT_ColumnFiltersState,
  MRT_PaginationState,
  MRT_SortingState,
  MRT_Updater,
} from 'material-react-table';
import { isSameTableState, mergePerconaTableState, resolveUpdater } from './tableState.utils';
import { type TableControlledState, type TableStateValues } from './tableState.types';
import {
  parseTableUrlState,
  serializeTableUrlState,
  hasActiveColumnFilters,
  resolveTableUrlSync,
  type TableUrlStateOptions,
  type TableUrlSyncKey,
} from './tableUrlState.utils';

export interface UsePerconaTableUrlStateOptions extends TableUrlStateOptions {
  searchParams: URLSearchParams;
  setSearchParams: (next: URLSearchParams, options?: { replace?: boolean }) => void;
  debounceMs?: number;
  replace?: boolean;
  additionalState?: Record<string, unknown>;
}

export type PerconaTableUrlControlledState = TableStateValues & {
  showColumnFilters: boolean;
  showGlobalFilter: boolean;
} & Record<string, unknown>;

export interface UsePerconaTableUrlStateResult {
  tableState: TableControlledState;
  tableProps: Pick<
    TableControlledState,
    | 'state'
    | 'onColumnFiltersChange'
    | 'onGlobalFilterChange'
    | 'onSortingChange'
    | 'onPaginationChange'
  > & {
    state: PerconaTableUrlControlledState;
    onShowColumnFiltersChange: (updater: MRT_Updater<boolean>) => void;
    onShowGlobalFilterChange: (updater: MRT_Updater<boolean>) => void;
  };
}

type FullTableUrlState = TableStateValues & {
  showColumnFilters: boolean;
  showGlobalFilter: boolean;
};

const DEFAULT_DEBOUNCE_MS = 300;

const isUrlSyncEnabled = (
  sync: Partial<Record<TableUrlSyncKey, boolean>> | undefined,
  key: TableUrlSyncKey
) => resolveTableUrlSync(sync)[key];

const mergeStateFromUrl = (
  prev: FullTableUrlState,
  searchParams: URLSearchParams,
  urlOptions: TableUrlStateOptions
): FullTableUrlState => {
  const parsed = parseTableUrlState(searchParams, urlOptions);
  const sync = resolveTableUrlSync(urlOptions.sync);
  const columnFilters = sync.filters ? parsed.columnFilters : prev.columnFilters;
  const globalFilter = sync.globalFilter ? parsed.globalFilter : prev.globalFilter;

  return {
    columnFilters,
    globalFilter,
    sorting: sync.sort ? parsed.sorting : prev.sorting,
    pagination: sync.pagination ? parsed.pagination : prev.pagination,
    showColumnFilters: hasActiveColumnFilters(columnFilters) ? true : prev.showColumnFilters,
    showGlobalFilter: globalFilter ? true : prev.showGlobalFilter,
  };
};

const toTableStateValues = ({
  columnFilters,
  globalFilter,
  sorting,
  pagination,
}: FullTableUrlState): TableStateValues => ({
  columnFilters,
  globalFilter,
  sorting,
  pagination,
});

const isSameFullTableUrlState = (a: FullTableUrlState, b: FullTableUrlState): boolean =>
  JSON.stringify(a) === JSON.stringify(b);

const createStateFromUrl = (
  searchParams: URLSearchParams,
  urlOptions: TableUrlStateOptions
): FullTableUrlState => {
  const parsed = parseTableUrlState(searchParams, urlOptions);
  return {
    ...parsed,
    showColumnFilters: hasActiveColumnFilters(parsed.columnFilters),
    showGlobalFilter: !!parsed.globalFilter,
  };
};

export function usePerconaTableUrlState({
  searchParams,
  setSearchParams,
  paramPrefix,
  defaults,
  sync,
  debounceMs = DEFAULT_DEBOUNCE_MS,
  replace = true,
  additionalState,
}: UsePerconaTableUrlStateOptions): UsePerconaTableUrlStateResult {
  const syncKey = JSON.stringify(sync ?? {});
  const defaultsKey = JSON.stringify(defaults ?? {});
  const additionalStateKey = JSON.stringify(additionalState ?? {});
  const urlOptions = useMemo(
    () => ({ paramPrefix, defaults, sync }),
    // defaults/sync objects are compared by serialized keys to avoid unstable deps
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [paramPrefix, defaultsKey, syncKey]
  );

  const searchParamsKey = searchParams.toString();

  const [state, setState] = useState<FullTableUrlState>(() =>
    createStateFromUrl(searchParams, urlOptions)
  );

  const globalFilterTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const columnFilterTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const isInternalUrlUpdateRef = useRef(false);
  const latestSearchParamsRef = useRef(searchParams);

  useEffect(() => {
    latestSearchParamsRef.current = searchParams;
  }, [searchParams, searchParamsKey]);

  useEffect(() => {
    if (isInternalUrlUpdateRef.current) {
      isInternalUrlUpdateRef.current = false;
      return;
    }

    setState((prev) => {
      const next = mergeStateFromUrl(prev, searchParams, urlOptions);
      return isSameFullTableUrlState(prev, next) ? prev : next;
    });
  }, [searchParams, searchParamsKey, urlOptions]);

  const writeUrl = useCallback(
    (nextState: TableStateValues) => {
      const currentParams = latestSearchParamsRef.current;
      const nextParams = serializeTableUrlState(nextState, currentParams, urlOptions);
      if (nextParams.toString() === currentParams.toString()) {
        return;
      }
      isInternalUrlUpdateRef.current = true;
      setSearchParams(nextParams, { replace });
    },
    [replace, setSearchParams, urlOptions]
  );

  const commitState = useCallback(
    (updater: (prev: TableStateValues) => TableStateValues, writeToUrl = true) => {
      setState((prev) => {
        const nextTableState = updater(toTableStateValues(prev));
        if (isSameTableState(toTableStateValues(prev), nextTableState)) {
          return prev;
        }
        const next = { ...prev, ...nextTableState };
        if (writeToUrl) {
          writeUrl(nextTableState);
        }
        return next;
      });
    },
    [writeUrl]
  );

  const onColumnFiltersChange = useCallback(
    (updater: MRT_Updater<MRT_ColumnFiltersState>) => {
      setState((prev) => {
        const nextTableState = {
          ...toTableStateValues(prev),
          columnFilters: resolveUpdater(updater, prev.columnFilters),
        };
        if (isSameTableState(toTableStateValues(prev), nextTableState)) {
          return prev;
        }
        const next = { ...prev, ...nextTableState };

        if (isUrlSyncEnabled(sync, 'filters')) {
          if (columnFilterTimerRef.current) {
            clearTimeout(columnFilterTimerRef.current);
          }
          columnFilterTimerRef.current = setTimeout(() => {
            setState((current) => {
              writeUrl(toTableStateValues(current));
              return current;
            });
          }, debounceMs);
        }

        return next;
      });
    },
    [debounceMs, sync, writeUrl]
  );

  const onSortingChange = useCallback(
    (updater: MRT_Updater<MRT_SortingState>) => {
      commitState(
        (prev) => ({ ...prev, sorting: resolveUpdater(updater, prev.sorting) }),
        isUrlSyncEnabled(sync, 'sort')
      );
    },
    [commitState, sync]
  );

  const onPaginationChange = useCallback(
    (updater: MRT_Updater<MRT_PaginationState>) => {
      commitState(
        (prev) => ({ ...prev, pagination: resolveUpdater(updater, prev.pagination) }),
        isUrlSyncEnabled(sync, 'pagination')
      );
    },
    [commitState, sync]
  );

  const onGlobalFilterChange = useCallback(
    (updater: MRT_Updater<string>) => {
      setState((prev) => {
        const nextGlobalFilter = resolveUpdater(updater, prev.globalFilter);
        if (nextGlobalFilter === prev.globalFilter) {
          return prev;
        }
        const next = { ...prev, globalFilter: nextGlobalFilter };

        if (isUrlSyncEnabled(sync, 'globalFilter')) {
          if (globalFilterTimerRef.current) {
            clearTimeout(globalFilterTimerRef.current);
          }

          globalFilterTimerRef.current = setTimeout(() => {
            setState((current) => {
              writeUrl(toTableStateValues(current));
              return current;
            });
          }, debounceMs);
        }

        return next;
      });
    },
    [debounceMs, sync, writeUrl]
  );

  useEffect(
    () => () => {
      if (globalFilterTimerRef.current) {
        clearTimeout(globalFilterTimerRef.current);
      }
      if (columnFilterTimerRef.current) {
        clearTimeout(columnFilterTimerRef.current);
      }
    },
    []
  );

  const onShowColumnFiltersChange = useCallback((updater: MRT_Updater<boolean>) => {
    setState((prev) => ({
      ...prev,
      showColumnFilters: resolveUpdater(updater, prev.showColumnFilters),
    }));
  }, []);

  const onShowGlobalFilterChange = useCallback((updater: MRT_Updater<boolean>) => {
    setState((prev) => ({
      ...prev,
      showGlobalFilter: resolveUpdater(updater, prev.showGlobalFilter),
    }));
  }, []);

  const tableValues = toTableStateValues(state);

  const tableState = useMemo<TableControlledState>(
    () => ({
      state: tableValues,
      onColumnFiltersChange,
      onGlobalFilterChange,
      onSortingChange,
      onPaginationChange,
    }),
    [tableValues, onColumnFiltersChange, onGlobalFilterChange, onSortingChange, onPaginationChange]
  );

  const tableProps = useMemo(
    () => ({
      ...tableState,
      state: mergePerconaTableState(
        {
          ...tableValues,
          showColumnFilters: state.showColumnFilters,
          showGlobalFilter: state.showGlobalFilter,
        },
        additionalState
      ),
      onShowColumnFiltersChange,
      onShowGlobalFilterChange,
    }),
    // additionalState is compared by serialized key to avoid unstable deps
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [
      tableState,
      tableValues,
      state.showColumnFilters,
      state.showGlobalFilter,
      additionalStateKey,
      onShowColumnFiltersChange,
      onShowGlobalFilterChange,
    ]
  );

  return {
    tableState,
    tableProps,
  };
}

export default usePerconaTableUrlState;
