import { describe, expect, it, vi } from 'vitest';
import {
  TEST_MONGO_DB_QUERY_DATA,
  TEST_MYSQL_QUERY_DATA,
} from 'utils/testStubs';
import { BlockedStatus, QueryData } from 'types/rta.types';
import {
  buildRtaExportFilename,
  collectCsvColumns,
  exportRtaQueriesToCsv,
  mapQueryToCsvRow,
  sanitizeCsvCell,
  toCsvHeader,
  toCsvValue,
} from './exportRtaQueriesToCsv';

const { download, generateCsv, mkConfig } = vi.hoisted(() => ({
  download: vi.fn(() => vi.fn()),
  generateCsv: vi.fn(() => vi.fn(() => 'csv-content')),
  mkConfig: vi.fn((config) => config),
}));

vi.mock('export-to-csv', () => ({
  download,
  generateCsv,
  mkConfig,
}));

const TEST_QUERY: QueryData = {
  ...TEST_MONGO_DB_QUERY_DATA,
  queryExecutionDurationMs: 10,
  mongoDbPayload: {
    ...TEST_MONGO_DB_QUERY_DATA.mongoDbPayload!,
    collection: 'mycollection',
  },
};

const TEST_MYSQL_QUERY: QueryData = {
  ...TEST_MYSQL_QUERY_DATA,
  queryExecutionDurationMs: 5,
};

describe('exportRtaQueriesToCsv', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('serializes values the csv generator cannot handle on its own', () => {
    expect(toCsvValue(10)).toBe(10);
    expect(toCsvValue(null)).toBe('');
    expect(toCsvValue(undefined)).toBe('');
    expect(toCsvValue(true)).toBe(true);
    expect(toCsvValue(['a', 'b'])).toBe('["a","b"]');
  });

  it('derives headers from the api field name unless it is overridden', () => {
    expect(toCsvHeader('queryId')).toBe('operation_id');
    expect(toCsvHeader('serviceName')).toBe('service');
    expect(toCsvHeader('dbInstanceAddress')).toBe('db_instance_address');
    expect(toCsvHeader('someNewApiField')).toBe('some_new_api_field');
  });

  it('sanitizes values that could be interpreted as spreadsheet formulas', () => {
    expect(sanitizeCsvCell('=1+1')).toBe("'=1+1");
    expect(sanitizeCsvCell('+cmd')).toBe("'+cmd");
    expect(sanitizeCsvCell('-10')).toBe("'-10");
    expect(sanitizeCsvCell('@SUM(A1)')).toBe("'@SUM(A1)");
    expect(sanitizeCsvCell('{ find: "x" }')).toBe('{ find: "x" }');
  });

  it('maps every query field to a csv column, flattening the payload', () => {
    expect(mapQueryToCsvRow(TEST_QUERY)).toEqual({
      operation_id: 'query-1',
      elapsed_exec_time_sec: 10,
      db_instance_address: '127.0.0.1',
      client_address: '127.0.0.1',
      database_name: 'database-name',
      service: 'Service 1',
      user_name: 'username',
      collection: 'mycollection',
      operation: 'operation',
      plan_summary: 'plan-summary',
      client_app_name: 'client-app-name',
      operation_start_time: '2021-01-01T00:00:00Z',
      data_capture_time: '2021-01-01T00:00:00Z',
      raw_query: '{ find: "mycollection", filter: { status: "active" } }',
      service_id: 'service-1',
      query_text: '{ find: "mycollection", filter: { status: "active" } }',
    });
  });

  it('maps every MySQL query field to a csv column, flattening the payload', () => {
    expect(mapQueryToCsvRow(TEST_MYSQL_QUERY)).toEqual({
      operation_id: 'query-2',
      elapsed_exec_time_sec: 5,
      db_instance_address: '127.0.0.1',
      client_address: '127.0.0.1',
      database_name: 'database-name',
      service: 'Service 2',
      user_name: 'username',
      command: 'Query',
      state: 'Sending data',
      program_name: 'mysql',
      rows_examined: 100,
      rows_sent: 10,
      full_scan: true,
      data_capture_time: '2021-01-01T00:00:00Z',
      raw_query: '{"current_statement": "SELECT * FROM my_table"}',
      service_id: 'service-2',
      query_text: 'SELECT * FROM my_table WHERE status = "active"',
    });
  });

  // The MySQL-specific fields are listed in the documented order, so the raw
  // JSON blob stays last instead of being pushed ahead of them.
  it('keeps the MySQL columns ahead of the raw query column', () => {
    const columns = collectCsvColumns([mapQueryToCsvRow(TEST_MYSQL_QUERY)]);

    expect(columns).toEqual([
      'operation_id',
      'elapsed_exec_time_sec',
      'db_instance_address',
      'client_address',
      'database_name',
      'service',
      'user_name',
      'command',
      'state',
      'program_name',
      'rows_examined',
      'rows_sent',
      'full_scan',
      'data_capture_time',
      'raw_query',
      'service_id',
      'query_text',
    ]);
  });

  it('keeps the documented column order and appends unlisted fields', () => {
    const columns = collectCsvColumns([mapQueryToCsvRow(TEST_QUERY)]);

    expect(columns).toEqual([
      'operation_id',
      'elapsed_exec_time_sec',
      'db_instance_address',
      'client_address',
      'database_name',
      'service',
      'user_name',
      'collection',
      'operation',
      'plan_summary',
      'client_app_name',
      'operation_start_time',
      'data_capture_time',
      'raw_query',
      'service_id',
      'query_text',
    ]);
  });

  it('exports fields the api adds without any mapping', () => {
    const row = mapQueryToCsvRow({
      ...TEST_QUERY,
      readPreference: 'secondary',
      mongoDbPayload: {
        ...TEST_QUERY.mongoDbPayload,
        waitingForLock: false,
        lockStats: { mode: 'IS' },
      },
    } as unknown as typeof TEST_QUERY);

    expect(row.read_preference).toBe('secondary');
    expect(row.waiting_for_lock).toBe(false);
    expect(row.mode).toBe('IS');
    expect(collectCsvColumns([row])).toContain('read_preference');
  });

  it('collects columns across all rows, not just the first', () => {
    const [withoutCollection, withCollection] = [
      mapQueryToCsvRow({
        ...TEST_QUERY,
        // The payload is optional now that a query may carry a MySQL one
        // instead, so the spread needs the assertion to stay fully typed.
        mongoDbPayload: {
          ...TEST_QUERY.mongoDbPayload!,
          collection: undefined,
        },
      }),
      mapQueryToCsvRow(TEST_QUERY),
    ];

    expect(collectCsvColumns([withoutCollection, withCollection])).toContain(
      'collection'
    );
  });

  it('builds the required filename template', () => {
    expect(
      buildRtaExportFilename(new Date('2026-06-25T14:30:22.000Z'))
    ).toMatch(/^rta_export_\d{8}_\d{6}$/);
  });

  it('exports filtered query rows to csv', () => {
    exportRtaQueriesToCsv([TEST_QUERY]);

    expect(mkConfig).toHaveBeenCalledWith({
      columnHeaders: expect.arrayContaining(['operation_id', 'raw_query']),
      // No technology prefix: RTA now covers MySQL as well as MongoDB.
      filename: expect.stringMatching(/^rta_export_\d{8}_\d{6}$/),
    });
    expect(generateCsv).toHaveBeenCalled();
    expect(download).toHaveBeenCalled();
  });

  it('does not export when there are no rows', () => {
    exportRtaQueriesToCsv([]);

    expect(mkConfig).not.toHaveBeenCalled();
    expect(generateCsv).not.toHaveBeenCalled();
    expect(download).not.toHaveBeenCalled();
  });
});

describe('blocking columns', () => {
  const blockedQuery = {
    ...TEST_MYSQL_QUERY,
    mySqlPayload: {
      ...TEST_MYSQL_QUERY.mySqlPayload!,
      blockedStatus: BlockedStatus.blocked,
      lockedTable: 'sbtest.sbtest1',
      lockedIndex: 'PRIMARY',
      blockedBy: [
        {
          blockingConnId: '412',
          blockingQuery: 'UPDATE sbtest1 SET k=k+1 WHERE id=1',
          blockingCommand: 'Query',
          blockingUsername: 'sbtest@172.17.0.1',
          root: false,
        },
        {
          blockingConnId: '409',
          blockingQuery: 'SELECT id,k FROM sbtest1 WHERE id=1 FOR UPDATE',
          blockingCommand: 'Sleep',
          blockingUsername: 'sbtest@172.17.0.1',
          root: true,
        },
      ],
    },
  };

  it('exports the head of the chain as plain columns', () => {
    const row = mapQueryToCsvRow(blockedQuery);

    expect(row.blocked_status).toBe('BLOCKED_STATUS_BLOCKED');
    expect(row.blocking_conn_id).toBe('409');
    // The contended lock belongs to the waiting statement, so it is one pair of columns.
    expect(row.locked_table).toBe('sbtest.sbtest1');
    expect(row.locked_index).toBe('PRIMARY');
    expect(row.blocking_query).toBe(
      'SELECT id,k FROM sbtest1 WHERE id=1 FOR UPDATE'
    );
  });

  it('never emits the blocker array as a JSON cell', () => {
    expect(mapQueryToCsvRow(blockedQuery)).not.toHaveProperty('blocked_by');
  });

  it('names nobody when several independent transactions are responsible', () => {
    const twoRoots = {
      ...blockedQuery,
      mySqlPayload: {
        ...blockedQuery.mySqlPayload!,
        blockedBy: blockedQuery.mySqlPayload!.blockedBy!.map((b) => ({
          ...b,
          root: true,
        })),
      },
    };
    const row = mapQueryToCsvRow(twoRoots);

    // Still findable as blocked, but no column claims one transaction is the cause.
    expect(row.blocked_status).toBe('BLOCKED_STATUS_BLOCKED');
    expect(row).not.toHaveProperty('blocking_conn_id');
  });

  it('leaves an unblocked statement without blocker columns', () => {
    const row = mapQueryToCsvRow(TEST_MYSQL_QUERY);

    expect(row).not.toHaveProperty('blocking_conn_id');
    expect(row).not.toHaveProperty('blocking_query');
  });
});
