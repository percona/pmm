import { describe, it, expect } from 'vitest';
import { type MRT_Row } from 'material-react-table';
import { BlockedStatus, QueryData, RawQueryData } from 'types/rta.types';
import {
  filterCommaSeparated,
  formatElapsedTime,
  isBlocked,
  isTransactionControl,
  queryDatabaseName,
  queryLanguage,
  queryUsername,
  blockingRoots,
  rtaRowId,
  soleBlocker,
  UNAVAILABLE_VALUE,
} from './OverviewTable.utils';
import {
  TEST_MONGO_DB_QUERY_DATA,
  TEST_MYSQL_QUERY_DATA,
} from 'utils/testStubs';

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
    [0, '0.000s'],
    [0.0004, '0.000s'],
    [0.003, '0.003s'],
    [0.0512, '0.051s'],
    [1.5234, '1.523s'],
    [9.9994, '9.999s'],
    [9.9996, '10s'],
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

// The lock graph observed for a three-connection pile-up: 409 is idle inside an open
// transaction and heads the chain, 412 is queued in the middle of it.
const blockedQuery: RawQueryData = {
  ...TEST_MYSQL_QUERY_DATA,
  mySqlPayload: {
    ...TEST_MYSQL_QUERY_DATA.mySqlPayload!,
    blockedStatus: BlockedStatus.blocked,
    blockedBy: [
      {
        blockingConnId: '409',
        blockingQuery: 'SELECT id,k FROM sbtest1 WHERE id=1 FOR UPDATE',
        blockingCommand: 'Sleep',
        blockingUsername: 'sbtest@172.17.0.1',
        root: true,
      },
      {
        blockingConnId: '412',
        blockingQuery: 'UPDATE sbtest1 SET k=k+1 WHERE id=1',
        blockingCommand: 'Query',
        blockingUsername: 'sbtest@172.17.0.1',
        root: false,
      },
    ],
  },
};

describe('isBlocked', () => {
  it('reports a MySQL statement waiting for a row lock', () => {
    expect(isBlocked(blockedQuery)).toBe(true);
  });

  it('reports an unblocked MySQL statement', () => {
    expect(isBlocked(TEST_MYSQL_QUERY_DATA)).toBe(false);
  });

  it('never reports a MongoDB statement as blocked', () => {
    expect(isBlocked(TEST_MONGO_DB_QUERY_DATA)).toBe(false);
  });
});

describe('soleBlocker', () => {
  it('names the head of the chain when exactly one transaction is responsible', () => {
    expect(soleBlocker(blockedQuery)?.blockingConnId).toBe('409');
  });

  it('names nobody when several independent transactions hold the statement up', () => {
    // Both blockers are non-waiting, so resolving either one leaves the statement blocked.
    const twoRoots: RawQueryData = {
      ...blockedQuery,
      mySqlPayload: {
        ...blockedQuery.mySqlPayload!,
        blockedBy: blockedQuery.mySqlPayload!.blockedBy!.map((blocker) => ({
          ...blocker,
          root: true,
        })),
      },
    };

    expect(soleBlocker(twoRoots)).toBeUndefined();
    expect(blockingRoots(twoRoots.mySqlPayload!.blockedBy!)).toHaveLength(2);
  });

  it('names nobody when the only blocker is itself waiting', () => {
    // root=false means the agent saw that connection waiting too, so resolving it is not
    // guaranteed to free this statement. The rule declines rather than guess.
    const cycle: RawQueryData = {
      ...blockedQuery,
      mySqlPayload: {
        ...blockedQuery.mySqlPayload!,
        blockedBy: [
          { ...blockedQuery.mySqlPayload!.blockedBy![0], root: false },
        ],
      },
    };

    expect(soleBlocker(cycle)).toBeUndefined();
  });

  it('names nobody in a lock cycle with several blockers', () => {
    const cycle: RawQueryData = {
      ...blockedQuery,
      mySqlPayload: {
        ...blockedQuery.mySqlPayload!,
        blockedBy: blockedQuery.mySqlPayload!.blockedBy!.map((blocker) => ({
          ...blocker,
          root: false,
        })),
      },
    };

    expect(soleBlocker(cycle)).toBeUndefined();
  });

  it('returns nothing for an unblocked statement', () => {
    expect(soleBlocker(TEST_MYSQL_QUERY_DATA)).toBeUndefined();
    expect(
      blockingRoots(TEST_MYSQL_QUERY_DATA.mySqlPayload?.blockedBy ?? [])
    ).toHaveLength(0);
  });
});

describe('rtaRowId', () => {
  it('keeps two instances apart when they hand out the same connection id', () => {
    // MySQL connection ids start at 1 and restart with the server, so two watched instances
    // share almost all of theirs. The details pane finds the selected row by matching this id.
    const onA: RawQueryData = {
      ...TEST_MYSQL_QUERY_DATA,
      serviceId: 'service-a',
      queryId: '411',
    };
    const onB: RawQueryData = {
      ...TEST_MYSQL_QUERY_DATA,
      serviceId: 'service-b',
      queryId: '411',
    };

    expect(rtaRowId(onA)).not.toBe(rtaRowId(onB));
  });

  it('is stable for the same row', () => {
    expect(rtaRowId(TEST_MYSQL_QUERY_DATA)).toBe(
      rtaRowId({ ...TEST_MYSQL_QUERY_DATA })
    );
  });

  it('still distinguishes rows within one service', () => {
    const a: RawQueryData = { ...TEST_MYSQL_QUERY_DATA, queryId: '411' };
    const b: RawQueryData = { ...TEST_MYSQL_QUERY_DATA, queryId: '412' };

    expect(rtaRowId(a)).not.toBe(rtaRowId(b));
  });
});
