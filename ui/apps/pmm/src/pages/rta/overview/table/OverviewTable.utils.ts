import { type MRT_Row } from 'material-react-table';
import { QueryData } from 'types/rta.types';

// What may sit in the field: a non-negative decimal, or a prefix of one. A
// lone '.' is on its way to '.5', so it has to be typeable.
const ELAPSED_TIME_INPUT = /^\d*\.?\d*$/;
// What the filter can actually use: a number of seconds. '' and '.' are not.
const ELAPSED_TIME_BOUND = /^(?:\d+(?:\.\d*)?|\.\d+)$/;

export const isElapsedTimeBoundInput = (value: string) =>
  ELAPSED_TIME_INPUT.test(value);

export const toElapsedTimeBound = (value: string) =>
  ELAPSED_TIME_BOUND.test(value) ? value : '';

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
