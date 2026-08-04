import { describe, it, expect } from 'vitest';
import { type MRT_Row } from 'material-react-table';
import { QueryData } from 'types/rta.types';
import {
  filterCommaSeparated,
  formatElapsedTime,
  isTransactionControl,
  queryDatabaseName,
  queryLanguage,
  queryUsername,
  UNAVAILABLE_VALUE,
} from './OverviewTable.utils';
import {
  TEST_MONGO_DB_QUERY_DATA,
  TEST_MYSQL_QUERY_DATA,
} from 'utils/testStubs';
import { RawQueryData } from 'types/rta.types';

const withText = (queryText: string): RawQueryData => ({
  ...TEST_MYSQL_QUERY_DATA,
  queryText,
});

describe('queryLanguage', () => {
  it('returns sql for MySQL queries', () => {
    expect(queryLanguage(TEST_MYSQL_QUERY_DATA)).toBe('sql');
  });

  it('returns mongodb for MongoDB queries', () => {
    expect(queryLanguage(TEST_MONGO_DB_QUERY_DATA)).toBe('mongodb');
  });
});

describe('queryDatabaseName', () => {
  it('resolves the database from the MongoDB payload', () => {
    expect(queryDatabaseName(TEST_MONGO_DB_QUERY_DATA)).toBe('database-name');
  });

  it('resolves the database from the MySQL payload', () => {
    expect(queryDatabaseName(TEST_MYSQL_QUERY_DATA)).toBe('database-name');
  });

  it('returns the unavailable label when the payload has no database', () => {
    expect(
      queryDatabaseName({
        ...TEST_MYSQL_QUERY_DATA,
        mySqlPayload: {
          ...TEST_MYSQL_QUERY_DATA.mySqlPayload!,
          databaseName: '',
        },
      })
    ).toBe(UNAVAILABLE_VALUE);
  });
});

describe('queryUsername', () => {
  it('resolves the user from the MongoDB payload', () => {
    expect(queryUsername(TEST_MONGO_DB_QUERY_DATA)).toBe('username');
  });

  it('resolves the user from the MySQL payload', () => {
    expect(queryUsername(TEST_MYSQL_QUERY_DATA)).toBe('username');
  });
});

describe('formatElapsedTime', () => {
  it.each([
    [0.0512, '0.1s'],
    [1.523, '1.5s'],
    [9.94, '9.9s'],
    [9.96, '10s'],
    [10.44, '10s'],
    [42.5, '43s'],
    [3601.7, '3602s'],
  ])('formats %j seconds as %j', (seconds, expected) => {
    expect(formatElapsedTime(seconds)).toBe(expected);
  });
});

describe('filterCommaSeparated', () => {
  const rowWithValue = (value: string) =>
    ({ getValue: () => value }) as unknown as MRT_Row<QueryData>;

  it('matches a single lazy (substring, case-insensitive) term', () => {
    expect(
      filterCommaSeparated(rowWithValue('sbtest'), 'databaseName', 'SBT')
    ).toBe(true);
    expect(
      filterCommaSeparated(rowWithValue('sbtest'), 'databaseName', 'orders')
    ).toBe(false);
  });

  it('matches any term of a comma-separated list', () => {
    expect(
      filterCommaSeparated(
        rowWithValue('orders'),
        'databaseName',
        'sbtest, ord'
      )
    ).toBe(true);
    expect(
      filterCommaSeparated(
        rowWithValue('inventory'),
        'databaseName',
        'sbtest, ord'
      )
    ).toBe(false);
  });

  it('ignores empty terms and passes everything for a blank filter', () => {
    expect(
      filterCommaSeparated(rowWithValue('anything'), 'databaseName', ' , ,')
    ).toBe(true);
    expect(
      filterCommaSeparated(rowWithValue('anything'), 'databaseName', '')
    ).toBe(true);
  });
});

describe('isTransactionControl', () => {
  it.each([
    'COMMIT',
    'commit',
    'COMMIT;',
    'COMMIT WORK',
    '  ROLLBACK  ',
    'ROLLBACK;',
    'BEGIN',
    'start transaction',
    'START  TRANSACTION',
  ])('flags transaction-control statement %j', (text) => {
    expect(isTransactionControl(withText(text))).toBe(true);
  });

  it.each([
    'SELECT 1',
    'UPDATE sbtest1 SET k=k+1',
    'COMMIT AND CHAIN',
    'INSERT INTO commits VALUES (1)',
  ])('does not flag regular statement %j', (text) => {
    expect(isTransactionControl(withText(text))).toBe(false);
  });
});
