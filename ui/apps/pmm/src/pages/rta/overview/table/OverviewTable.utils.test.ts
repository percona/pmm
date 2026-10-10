import { describe, it, expect } from 'vitest';
import { type MRT_Row } from 'material-react-table';
import { BlockedStatus, QueryData, RawQueryData } from 'types/rta.types';
import {
  filterCommaSeparated,
  filterElapsedTime,
  formatElapsedTime,
  formatLockTimeMs,
  lockTimeMs,
  isBlocked,
  isBlockingUnattributed,
  isBlockingUnknown,
  isElapsedTimeBoundInput,
  isSameStatement,
  isTransactionControl,
  queryDatabaseName,
  queryLanguage,
  queryUsername,
  blockingRoots,
  rtaRowId,
  soleBlocker,
  statementRowId,
  toElapsedTimeBound,
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

describe('formatLockTimeMs', () => {
  it('renders microsecond lock waits in milliseconds', () => {
    // 3000000 ps from LOCK_TIME arrives as "0.000003s".
    expect(formatLockTimeMs('0.000003s')).toBe('0.003 ms');
    expect(formatLockTimeMs('0.001250s')).toBe('1.25 ms');
    expect(formatLockTimeMs('2s')).toBe('2000 ms');
    expect(formatLockTimeMs('0s')).toBe('0 ms');
  });

  it('leaves an unmeasured lock time blank', () => {
    expect(formatLockTimeMs(undefined)).toBe('');
    expect(formatLockTimeMs(null)).toBe('');
    expect(lockTimeMs(undefined)).toBeUndefined();
  });
});

describe('isBlockingUnattributed', () => {
  const withStatus = (blockedStatus?: BlockedStatus): RawQueryData => ({
    ...TEST_MYSQL_QUERY_DATA,
    mySqlPayload: { ...TEST_MYSQL_QUERY_DATA.mySqlPayload!, blockedStatus },
  });

  it('is true only for a wait that belonged to a later statement', () => {
    expect(isBlockingUnattributed(withStatus(BlockedStatus.unattributed))).toBe(
      true
    );
    expect(isBlockingUnattributed(withStatus(BlockedStatus.unspecified))).toBe(
      false
    );
    expect(isBlockingUnattributed(withStatus(BlockedStatus.blocked))).toBe(
      false
    );
    expect(isBlockingUnattributed(TEST_MONGO_DB_QUERY_DATA)).toBe(false);
  });

  it('is not mistaken for an unreadable lock source', () => {
    const row = withStatus(BlockedStatus.unattributed);

    expect(isBlockingUnknown(row)).toBe(false);
    expect(isBlocked(row)).toBe(false);
  });
});

describe('isSameStatement', () => {
  const running: QueryData = {
    ...TEST_MYSQL_QUERY_DATA,
    queryText: 'SELECT SLEEP(100)',
    queryExecutionDurationMs: 10_000,
  };

  it('follows a statement that is still running', () => {
    expect(
      isSameStatement(running, { ...running, queryExecutionDurationMs: 12_000 })
    ).toBe(true);
  });

  it('does not mistake the connection running something else for it', () => {
    expect(isSameStatement(running, { ...running, queryText: 'COMMIT' })).toBe(
      false
    );
  });

  it('does not mistake a later run of the same text for it', () => {
    expect(
      isSameStatement(running, { ...running, queryExecutionDurationMs: 500 })
    ).toBe(false);
  });

  it('keys navigation on the statement, not only the connection', () => {
    expect(statementRowId(running)).not.toBe(
      statementRowId({ ...running, queryText: 'COMMIT' })
    );
  });
});

const row = (seconds: number | null) =>
  ({ getValue: () => seconds }) as unknown as MRT_Row<QueryData>;

const ID = 'queryExecutionDurationMs';

describe('isElapsedTimeBoundInput', () => {
  it.each(['', '.', '0', '1', '30', '1.', '.5', '1.5', '1.50'])(
    'accepts %s',
    (value) => {
      expect(isElapsedTimeBoundInput(value)).toBe(true);
    }
  );

  it.each(['abc', '1a', 'a1', '1.5x', ' 1', '1 ', '-1', '+1', '1e3', '1.2.3'])(
    'rejects %s',
    (value) => {
      expect(isElapsedTimeBoundInput(value)).toBe(false);
    }
  );
});

describe('toElapsedTimeBound', () => {
  it.each(['0', '1', '30', '1.', '.5', '1.5', '1.50'])('keeps %s', (value) => {
    expect(toElapsedTimeBound(value)).toBe(value);
  });

  // '' and '.' are legal to be typing but are not a number of seconds, and
  // parseFloat is too lenient to tell '1.5x' apart from a bound.
  it.each(['', '.', 'abc', '1.5x', '-1', '1e3', '1.2.3'])(
    'drops %s',
    (value) => {
      expect(toElapsedTimeBound(value)).toBe('');
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
