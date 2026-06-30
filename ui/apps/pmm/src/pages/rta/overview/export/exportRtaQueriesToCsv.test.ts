import { describe, expect, it, vi } from 'vitest';
import { TEST_MONGO_DB_QUERY_DATA } from 'utils/testStubs';
import {
  buildRtaExportFilename,
  exportRtaQueriesToCsv,
  formatElapsedTimeForExport,
  mapQueryToCsvRow,
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

describe('exportRtaQueriesToCsv', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('formats elapsed time like the overview table', () => {
    expect(formatElapsedTimeForExport(10)).toBe('10 seconds');
    expect(formatElapsedTimeForExport(null)).toBe('');
  });

  it('maps query data to csv row columns', () => {
    expect(
      mapQueryToCsvRow({
        ...TEST_MONGO_DB_QUERY_DATA,
        queryExecutionDurationMs: 10,
      })
    ).toEqual({
      'Operation ID': 'query-1',
      Service: 'Service 1',
      'Query Text':
        '{ find: "mycollection", filter: { status: "active" } }',
      'Elapsed Time': '10 seconds',
      'Plan Summary': 'plan-summary',
    });
  });

  it('builds the required filename template', () => {
    expect(buildRtaExportFilename(new Date('2026-06-25T14:30:22.000Z'))).toMatch(
      /^mongodb_rta_export_\d{8}_\d{6}$/
    );
  });

  it('exports filtered query rows to csv', () => {
    const query = {
      ...TEST_MONGO_DB_QUERY_DATA,
      queryExecutionDurationMs: 10,
    };

    exportRtaQueriesToCsv([query]);

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
