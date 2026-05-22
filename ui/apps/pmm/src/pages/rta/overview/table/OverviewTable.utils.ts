import { type MRT_Row } from 'material-react-table';
import { QueryData } from 'types/rta.types';

export const getNavigableQueryIdsKey = (queries: QueryData[]) =>
  queries.map((query) => query.queryId).join('\0');

export const isSameTableState = <T>(previous: T, next: T) =>
  JSON.stringify(previous) === JSON.stringify(next);

export const resolveTableStateUpdate = <T>(
  previous: T,
  updater: T | ((old: T) => T)
): T => {
  if (typeof updater === 'function') {
    return (updater as (old: T) => T)(previous);
  }
  return updater;
};

export const filterElapsedTime = (
  row: MRT_Row<QueryData>,
  id: string,
  filterValue: [string, string]
) => {
  const [min, max] = filterValue;
  const valueSeconds = row.getValue<number>(id);
  if (valueSeconds === null || valueSeconds === undefined) return false;

  const minSet = min !== '' && min != null && !Number.isNaN(parseFloat(min));
  const maxSet = max !== '' && max != null && !Number.isNaN(parseFloat(max));

  if (!minSet && !maxSet) return true;

  if (minSet && !maxSet) return valueSeconds >= parseFloat(min);
  if (!minSet && maxSet) return valueSeconds <= parseFloat(max);

  return valueSeconds >= parseFloat(min) && valueSeconds <= parseFloat(max);
};
