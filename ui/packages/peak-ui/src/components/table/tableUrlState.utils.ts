import { MRT_ColumnFiltersState, MRT_SortingState } from 'material-react-table';
import { DEFAULT_TABLE_STATE, type TableStateValues } from './tableState.types';

export type TableUrlSyncKey = 'filters' | 'sort' | 'globalFilter' | 'pagination';

export interface TableUrlStateOptions {
  paramPrefix?: string;
  defaults?: Partial<TableStateValues>;
  sync?: Partial<Record<TableUrlSyncKey, boolean>>;
}

const DEFAULT_SYNC: Record<TableUrlSyncKey, boolean> = {
  filters: true,
  sort: true,
  globalFilter: true,
  pagination: true,
};

export const resolveTableUrlSync = (
  sync?: Partial<Record<TableUrlSyncKey, boolean>>
): Record<TableUrlSyncKey, boolean> => ({
  ...DEFAULT_SYNC,
  ...sync,
});

const paramKey = (base: string, prefix?: string) => (prefix ? `${prefix}.${base}` : base);

const filterParamPrefix = (prefix?: string) => paramKey('f', prefix);

const resolveTableUrlDefaults = (defaults?: Partial<TableStateValues>): TableStateValues => ({
  ...DEFAULT_TABLE_STATE,
  ...defaults,
  pagination: {
    ...DEFAULT_TABLE_STATE.pagination,
    ...defaults?.pagination,
  },
});

export const buildTableUrlParamKeys = (prefix?: string) => ({
  globalFilter: paramKey('q', prefix),
  sort: paramKey('sort', prefix),
  page: paramKey('page', prefix),
  pageSize: paramKey('pageSize', prefix),
  filterPrefix: `${filterParamPrefix(prefix)}.`,
});

const parsePositiveInt = (value: string | null, fallback: number) => {
  if (!value) {
    return fallback;
  }
  const parsed = Number.parseInt(value, 10);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback;
};

const parseSorting = (value: string | null): MRT_SortingState => {
  if (!value) {
    return [];
  }

  return value
    .split(',')
    .map((part) => part.trim())
    .filter(Boolean)
    .map((part) => {
      const [id, direction] = part.split(':');
      if (!id) {
        return null;
      }
      return { id, desc: direction === 'desc' };
    })
    .filter((entry): entry is { id: string; desc: boolean } => entry !== null);
};

const normalizeFilterArray = (value: unknown[]): unknown[] =>
  value.map((entry) => (entry === null || entry === undefined ? '' : entry));

const parseFilterValue = (raw: string): unknown => {
  if (raw.startsWith('[')) {
    try {
      const parsed = JSON.parse(raw);
      if (Array.isArray(parsed)) {
        return normalizeFilterArray(parsed);
      }
    } catch {
      // fall through to scalar
    }
  }
  return raw;
};

const serializeFilterValue = (value: unknown): string =>
  Array.isArray(value) ? JSON.stringify(normalizeFilterArray(value)) : String(value);

const isEmptyFilterValue = (value: unknown): boolean => {
  if (value === undefined || value === null || value === '') {
    return true;
  }
  if (Array.isArray(value)) {
    return value.every((entry) => entry === '' || entry === undefined || entry === null);
  }
  return false;
};

export const hasActiveColumnFilters = (filters: MRT_ColumnFiltersState): boolean =>
  filters.some(({ value }) => !isEmptyFilterValue(value));

const parseColumnFilters = (
  searchParams: URLSearchParams,
  filterPrefix: string
): MRT_ColumnFiltersState => {
  const filters: MRT_ColumnFiltersState = [];

  searchParams.forEach((value, key) => {
    if (!key.startsWith(filterPrefix) || !value) {
      return;
    }
    const id = key.slice(filterPrefix.length);
    if (!id) {
      return;
    }
    filters.push({ id, value: parseFilterValue(value) });
  });

  return filters;
};

const hasFilterParams = (searchParams: URLSearchParams, filterPrefix: string): boolean =>
  Array.from(searchParams.keys()).some((key) => key.startsWith(filterPrefix));

const explicitEmptyFiltersKey = (filterPrefix: string) => `${filterPrefix}_`;

const parseColumnFiltersFromUrl = (
  searchParams: URLSearchParams,
  filterPrefix: string
): MRT_ColumnFiltersState => {
  if (searchParams.has(explicitEmptyFiltersKey(filterPrefix))) {
    const filters = parseColumnFilters(searchParams, filterPrefix).filter(({ id }) => id !== '_');
    return hasActiveColumnFilters(filters) ? filters : [];
  }

  return parseColumnFilters(searchParams, filterPrefix);
};

export const parseTableUrlState = (
  searchParams: URLSearchParams,
  options: TableUrlStateOptions = {}
): TableStateValues => {
  const sync = resolveTableUrlSync(options.sync);
  const defaults = resolveTableUrlDefaults(options.defaults);
  const keys = buildTableUrlParamKeys(options.paramPrefix);

  const pageFromUrl = sync.pagination
    ? parsePositiveInt(searchParams.get(keys.page), defaults.pagination.pageIndex + 1)
    : defaults.pagination.pageIndex + 1;
  const pageSize = sync.pagination
    ? parsePositiveInt(searchParams.get(keys.pageSize), defaults.pagination.pageSize)
    : defaults.pagination.pageSize;

  return {
    columnFilters: sync.filters
      ? hasFilterParams(searchParams, keys.filterPrefix)
        ? parseColumnFiltersFromUrl(searchParams, keys.filterPrefix)
        : defaults.columnFilters
      : defaults.columnFilters,
    globalFilter: sync.globalFilter
      ? searchParams.has(keys.globalFilter)
        ? (searchParams.get(keys.globalFilter) ?? '')
        : defaults.globalFilter
      : defaults.globalFilter,
    sorting: sync.sort
      ? searchParams.has(keys.sort)
        ? parseSorting(searchParams.get(keys.sort))
        : defaults.sorting
      : defaults.sorting,
    pagination: sync.pagination
      ? { pageIndex: Math.max(0, pageFromUrl - 1), pageSize }
      : defaults.pagination,
  };
};

const serializeSorting = (sorting: MRT_SortingState) =>
  sorting.map(({ id, desc }) => `${id}:${desc ? 'desc' : 'asc'}`).join(',');

const removeManagedParams = (
  searchParams: URLSearchParams,
  keys: ReturnType<typeof buildTableUrlParamKeys>,
  sync: Record<TableUrlSyncKey, boolean>
) => {
  const next = new URLSearchParams(searchParams);

  Array.from(next.keys()).forEach((key) => {
    if (sync.globalFilter && key === keys.globalFilter) {
      next.delete(key);
    }
    if (sync.sort && key === keys.sort) {
      next.delete(key);
    }
    if (sync.pagination && (key === keys.page || key === keys.pageSize)) {
      next.delete(key);
    }
    if (sync.filters && key.startsWith(keys.filterPrefix)) {
      next.delete(key);
    }
  });

  return next;
};

export const serializeTableUrlState = (
  state: TableStateValues,
  searchParams: URLSearchParams,
  options: TableUrlStateOptions = {}
): URLSearchParams => {
  const sync = resolveTableUrlSync(options.sync);
  const defaults = resolveTableUrlDefaults(options.defaults);
  const keys = buildTableUrlParamKeys(options.paramPrefix);
  const next = removeManagedParams(searchParams, keys, sync);

  if (sync.globalFilter && state.globalFilter !== defaults.globalFilter) {
    next.set(keys.globalFilter, state.globalFilter);
  }

  if (sync.sort) {
    const serializedSorting = serializeSorting(state.sorting);
    const defaultSorting = serializeSorting(defaults.sorting);
    if (serializedSorting !== defaultSorting) {
      next.set(keys.sort, serializedSorting);
    }
  }

  if (sync.filters) {
    if (hasActiveColumnFilters(state.columnFilters)) {
      state.columnFilters.forEach(({ id, value }) => {
        if (isEmptyFilterValue(value)) {
          return;
        }
        next.set(`${keys.filterPrefix}${id}`, serializeFilterValue(value));
      });
    } else if (hasActiveColumnFilters(defaults.columnFilters)) {
      next.set(explicitEmptyFiltersKey(keys.filterPrefix), '1');
    }
  }

  if (sync.pagination) {
    const pageIndex = state.pagination.pageIndex + 1;
    if (pageIndex !== defaults.pagination.pageIndex + 1) {
      next.set(keys.page, String(pageIndex));
    }
    if (state.pagination.pageSize !== defaults.pagination.pageSize) {
      next.set(keys.pageSize, String(state.pagination.pageSize));
    }
  }

  return next;
};

export const normalizeSearchParamsKey = (params: URLSearchParams): string =>
  JSON.stringify(
    Array.from(params.entries()).sort(([aKey, aVal], [bKey, bVal]) =>
      aKey === bKey ? aVal.localeCompare(bVal) : aKey.localeCompare(bKey)
    )
  );
