import { MRT_Updater } from 'material-react-table';
import { type TableStateValues } from './tableState.types';

export const resolveUpdater = <S>(updater: MRT_Updater<S>, prev: S): S =>
  updater instanceof Function ? updater(prev) : updater;

export const isSameTableState = (a: TableStateValues, b: TableStateValues): boolean =>
  JSON.stringify(a) === JSON.stringify(b);

export const stableDependencyKey = (value: unknown): string => {
  if (value === undefined || value === null) {
    return '';
  }

  try {
    return JSON.stringify(value);
  } catch {
    if (typeof value === 'object') {
      return Object.entries(value as Record<string, unknown>)
        .map(([key, entry]) => {
          try {
            return `${key}:${JSON.stringify(entry)}`;
          } catch {
            return key;
          }
        })
        .sort()
        .join('|');
    }

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
