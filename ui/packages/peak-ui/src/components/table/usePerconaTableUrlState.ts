import { useCallback, useEffect, useMemo, useRef, useState, type MutableRefObject } from 'react';
import {
  MRT_ColumnFiltersState,
  MRT_PaginationState,
  MRT_SortingState,
  MRT_Updater,
} from 'material-react-table';
import {
  isSameTableState,
  mergePerconaTableState,
  resolveUpdater,
  stableDependencyKey,
} from './tableState.utils';
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

const clearDebouncedUrlWriteTimers = (
  columnFilterTimerRef: MutableRefObject<ReturnType<typeof setTimeout> | null>,
  globalFilterTimerRef: MutableRefObject<ReturnType<typeof setTimeout> | null>
) => {
  if (columnFilterTimerRef.current) {
    clearTimeout(columnFilterTimerRef.current);
    columnFilterTimerRef.current = null;
  }
  if (globalFilterTimerRef.current) {
    clearTimeout(globalFilterTimerRef.current);
    globalFilterTimerRef.current = null;
  }
};

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
  stableDependencyKey(a) === stableDependencyKey(b);

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
  const syncKey = stableDependencyKey(sync);
  const defaultsKey = stableDependencyKey(defaults);
  const additionalStateKey = stableDependencyKey(additionalState);
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
  const latestTableStateRef = useRef(state);

  useEffect(() => {
    latestTableStateRef.current = state;
  }, [state]);

  useEffect(() => {
    latestSearchParamsRef.current = searchParams;
  }, [searchParams, searchParamsKey]);

  useEffect(() => {
    if (isInternalUrlUpdateRef.current) {
      isInternalUrlUpdateRef.current = false;
      return;
    }

    clearDebouncedUrlWriteTimers(columnFilterTimerRef, globalFilterTimerRef);

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

  const scheduleDebouncedUrlWrite = useCallback(
    (timerRef: MutableRefObject<ReturnType<typeof setTimeout> | null>) => {
      if (timerRef.current) {
        clearTimeout(timerRef.current);
      }
      timerRef.current = setTimeout(() => {
        writeUrl(toTableStateValues(latestTableStateRef.current));
      }, debounceMs);
    },
    [debounceMs, writeUrl]
  );

  const commitState = useCallback(
    (updater: (prev: TableStateValues) => TableStateValues, writeToUrl = true) => {
      const prev = latestTableStateRef.current;
      const nextTableState = updater(toTableStateValues(prev));
      if (isSameTableState(toTableStateValues(prev), nextTableState)) {
        return;
      }
      const next = { ...prev, ...nextTableState };
      latestTableStateRef.current = next;
      setState(next);
      if (writeToUrl) {
        writeUrl(nextTableState);
      }
    },
    [writeUrl]
  );

  const onColumnFiltersChange = useCallback(
    (updater: MRT_Updater<MRT_ColumnFiltersState>) => {
      const prev = latestTableStateRef.current;
      const nextTableState = {
        ...toTableStateValues(prev),
        columnFilters: resolveUpdater(updater, prev.columnFilters),
      };
      if (isSameTableState(toTableStateValues(prev), nextTableState)) {
        return;
      }
      const next = { ...prev, ...nextTableState };
      latestTableStateRef.current = next;
      setState(next);

      if (isUrlSyncEnabled(sync, 'filters')) {
        scheduleDebouncedUrlWrite(columnFilterTimerRef);
      }
    },
    [scheduleDebouncedUrlWrite, sync]
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
      const prev = latestTableStateRef.current;
      const nextGlobalFilter = resolveUpdater(updater, prev.globalFilter);
      if (nextGlobalFilter === prev.globalFilter) {
        return;
      }
      const next = { ...prev, globalFilter: nextGlobalFilter };
      latestTableStateRef.current = next;
      setState(next);

      if (isUrlSyncEnabled(sync, 'globalFilter')) {
        scheduleDebouncedUrlWrite(globalFilterTimerRef);
      }
    },
    [scheduleDebouncedUrlWrite, sync]
  );

  useEffect(
    () => () => {
      clearDebouncedUrlWriteTimers(columnFilterTimerRef, globalFilterTimerRef);
    },
    []
  );

  const onShowColumnFiltersChange = useCallback((updater: MRT_Updater<boolean>) => {
    const prev = latestTableStateRef.current;
    const next = {
      ...prev,
      showColumnFilters: resolveUpdater(updater, prev.showColumnFilters),
    };
    latestTableStateRef.current = next;
    setState(next);
  }, []);

  const onShowGlobalFilterChange = useCallback((updater: MRT_Updater<boolean>) => {
    const prev = latestTableStateRef.current;
    const next = {
      ...prev,
      showGlobalFilter: resolveUpdater(updater, prev.showGlobalFilter),
    };
    latestTableStateRef.current = next;
    setState(next);
  }, []);

  const tableValues = useMemo(() => toTableStateValues(state), [state]);

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
