import { describe, it, expect } from 'vitest';
import {
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
