import { describe, it, expect } from 'vitest';
import { isTransactionControl, queryLanguage } from './OverviewTable.utils';
import { TEST_MONGO_DB_QUERY_DATA, TEST_MYSQL_QUERY_DATA } from 'utils/testStubs';
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

describe('isTransactionControl', () => {
  it.each(['COMMIT', 'commit', '  ROLLBACK  ', 'BEGIN', 'start transaction'])(
    'flags transaction-control statement %j',
    (text) => {
      expect(isTransactionControl(withText(text))).toBe(true);
    }
  );

  it.each(['SELECT 1', 'UPDATE sbtest1 SET k=k+1', 'COMMIT AND CHAIN'])(
    'does not flag regular statement %j',
    (text) => {
      expect(isTransactionControl(withText(text))).toBe(false);
    }
  );
});
