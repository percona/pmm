import type { MRT_Row } from 'material-react-table';
import type { QueryData } from 'types/rta.types';
import { filterElapsedTime } from './OverviewTable.utils';

const row = (seconds: number | null) =>
  ({ getValue: () => seconds }) as unknown as MRT_Row<QueryData>;

const ID = 'queryExecutionDurationMs';

describe('filterElapsedTime', () => {
  it('keeps every row when neither bound is set', () => {
    expect(filterElapsedTime(row(1.5), ID, ['', ''])).toBe(true);
  });

  it('applies the min bound alone', () => {
    expect(filterElapsedTime(row(1.5), ID, ['1.5', ''])).toBe(true);
    expect(filterElapsedTime(row(1.4), ID, ['1.5', ''])).toBe(false);
  });

  it('applies the max bound alone', () => {
    expect(filterElapsedTime(row(1.5), ID, ['', '1.5'])).toBe(true);
    expect(filterElapsedTime(row(1.6), ID, ['', '1.5'])).toBe(false);
  });

  it('applies both bounds inclusively', () => {
    expect(filterElapsedTime(row(5), ID, ['1', '10'])).toBe(true);
    expect(filterElapsedTime(row(11), ID, ['1', '10'])).toBe(false);
  });

  it('drops rows without an elapsed time', () => {
    expect(filterElapsedTime(row(null), ID, ['', ''])).toBe(false);
  });
});
