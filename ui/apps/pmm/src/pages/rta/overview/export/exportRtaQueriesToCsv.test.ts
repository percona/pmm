import { describe, expect, it, vi } from 'vitest';
import { TEST_MONGO_DB_QUERY_DATA } from 'utils/testStubs';
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

const TEST_QUERY = {
  ...TEST_MONGO_DB_QUERY_DATA,
  queryExecutionDurationMs: 10,
  mongoDbPayload: {
    ...TEST_MONGO_DB_QUERY_DATA.mongoDbPayload,
    collection: 'mycollection',
  },
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
        mongoDbPayload: {
          ...TEST_QUERY.mongoDbPayload,
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
    ).toMatch(/^mongodb_rta_export_\d{8}_\d{6}$/);
  });

  it('exports filtered query rows to csv', () => {
    exportRtaQueriesToCsv([TEST_QUERY]);

    expect(mkConfig).toHaveBeenCalledWith({
      columnHeaders: expect.arrayContaining(['operation_id', 'raw_query']),
      filename: expect.stringMatching(/^mongodb_rta_export_\d{8}_\d{6}$/),
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
