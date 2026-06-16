import { type MRT_Row } from 'material-react-table';
import { QueryData, RawQueryData } from 'types/rta.types';
import { CodeLanguage } from 'types/util.types';

// queryLanguage returns the syntax-highlighting language for a query
// based on which database-specific payload it carries.
export const queryLanguage = (query: RawQueryData): CodeLanguage =>
  query.mySqlPayload ? 'sql' : 'mongodb';

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
