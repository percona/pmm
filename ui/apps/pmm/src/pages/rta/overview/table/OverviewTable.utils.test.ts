import type { MRT_Row } from 'material-react-table';
import type { QueryData } from 'types/rta.types';
import { filterElapsedTime, isElapsedTimeBound } from './OverviewTable.utils';

const row = (seconds: number | null) =>
  ({ getValue: () => seconds }) as unknown as MRT_Row<QueryData>;

const ID = 'queryExecutionDurationMs';

describe('isElapsedTimeBound', () => {
  it.each(['', '0', '1', '30', '1.', '.5', '1.5', '1.50'])(
    'accepts %s',
    (value) => {
      expect(isElapsedTimeBound(value)).toBe(true);
    }
  );

  it.each(['abc', '1a', 'a1', '1.5x', ' 1', '1 ', '-1', '+1', '1e3', '1.2.3'])(
    'rejects %s',
    (value) => {
      expect(isElapsedTimeBound(value)).toBe(false);
    }
  );
});

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
