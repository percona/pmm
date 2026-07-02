import { describe, expect, it, vi } from 'vitest';
import { TEST_MONGO_DB_QUERY_DATA } from 'utils/testStubs';
import {
  buildRtaExportFilename,
  exportRtaQueriesToCsv,
  formatElapsedExecTimeSec,
  mapQueryToCsvRow,
  sanitizeCsvCell,
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

  it('formats elapsed exec time as seconds', () => {
    expect(formatElapsedExecTimeSec(10)).toBe(10);
    expect(formatElapsedExecTimeSec(null)).toBe('');
  });

  it('sanitizes values that could be interpreted as spreadsheet formulas', () => {
    expect(sanitizeCsvCell('=1+1')).toBe("'=1+1");
    expect(sanitizeCsvCell('+cmd')).toBe("'+cmd");
    expect(sanitizeCsvCell('-10')).toBe("'-10");
    expect(sanitizeCsvCell('@SUM(A1)')).toBe("'@SUM(A1)");
    expect(sanitizeCsvCell('{ find: "x" }')).toBe('{ find: "x" }');
  });

  it('maps query data to csv row columns in the required order', () => {
    const row = mapQueryToCsvRow(TEST_QUERY);

    expect(Object.keys(row)).toEqual([
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
    ]);

    expect(row).toEqual({
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
    });
  });

  it('builds the required filename template', () => {
    expect(buildRtaExportFilename(new Date('2026-06-25T14:30:22.000Z'))).toMatch(
      /^mongodb_rta_export_\d{8}_\d{6}$/
    );
  });

  it('exports filtered query rows to csv', () => {
    exportRtaQueriesToCsv([TEST_QUERY]);

    expect(mkConfig).toHaveBeenCalledWith({
      useKeysAsHeaders: true,
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
