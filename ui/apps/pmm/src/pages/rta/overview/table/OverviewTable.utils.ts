import { type MRT_Row } from 'material-react-table';
import { QueryData } from 'types/rta.types';

// An elapsed time bound is a non-negative number of seconds. The empty string
// and partial values like '1.' or '.5' pass so the user can keep typing.
const ELAPSED_TIME_BOUND = /^\d*\.?\d*$/;

export const isElapsedTimeBound = (value: string) =>
  ELAPSED_TIME_BOUND.test(value);

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
