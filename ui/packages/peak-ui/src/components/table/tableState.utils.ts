import { MRT_ColumnFiltersState, MRT_Updater } from 'material-react-table';
import { type TableStateValues } from './tableState.types';

export const resolveUpdater = <S>(updater: MRT_Updater<S>, prev: S): S =>
  updater instanceof Function ? updater(prev) : updater;

// MRT range filters mutate value arrays in place; clone at controlled-state boundaries.
export const cloneColumnFilters = (
  filters: MRT_ColumnFiltersState
): MRT_ColumnFiltersState =>
  filters.map(({ id, value }) => ({
    id,
    value: Array.isArray(value) ? [...value] : value,
  }));

export const isSameTableState = (a: TableStateValues, b: TableStateValues): boolean =>
  stableDependencyKey(a) === stableDependencyKey(b);

export const stableDependencyKey = (value: unknown): string => {
  if (value === undefined || value === null) {
    return '';
  }

  const seen = new WeakSet<object>();

  const normalize = (input: unknown): unknown => {
    if (typeof input === 'bigint') {
      return input.toString();
    }
    if (typeof input === 'function') {
      return '[function]';
    }
    if (typeof input === 'symbol') {
      return input.toString();
    }
    if (!input || typeof input !== 'object') {
      return input;
    }

    const obj = input as Record<string, unknown>;
    if (seen.has(obj)) {
      return '[circular]';
    }
    seen.add(obj);

    if (Array.isArray(input)) {
      return input.map(normalize);
    }

    return Object.keys(obj)
      .sort()
      .reduce<Record<string, unknown>>((acc, key) => {
        acc[key] = normalize(obj[key]);
        return acc;
      }, {});
  };

  try {
    return JSON.stringify(normalize(value)) ?? '';
  } catch {
    return String(value);
  }
};

export const mergePerconaTableState = <
  TUrlState extends TableStateValues & {
    showColumnFilters?: boolean;
    showGlobalFilter?: boolean;
  },
>(
  urlState: TUrlState,
  additionalState?: Record<string, unknown>
): TUrlState & Record<string, unknown> => ({
  ...(additionalState ?? {}),
  ...urlState,
});
