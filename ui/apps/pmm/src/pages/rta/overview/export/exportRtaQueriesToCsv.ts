import { format } from 'date-fns';
import { download, generateCsv, mkConfig } from 'export-to-csv';
import type { QueryData } from 'types/rta.types';
import { isPlainObject } from 'utils/object.utils';

const CSV_FORMULA_PREFIX = /^[=+\-@\t\r]/;

// Headers that differ from the API field name. Every other field falls back to
// the snake_case form of its key, which round-trips back to the name the API
// returned before axios-case-converter camelized it.
const CSV_HEADER_OVERRIDES: Record<string, string> = {
  queryId: 'operation_id',
  serviceName: 'service',
  queryExecutionDurationMs: 'elapsed_exec_time_sec',
  username: 'user_name',
  queryCollectTime: 'data_capture_time',
  queryRawJson: 'raw_query',
};

type CsvValue = string | number | boolean;
type CsvRow = Record<string, CsvValue>;

export const sanitizeCsvCell = (value: string): string => {
  if (CSV_FORMULA_PREFIX.test(value)) {
    return `'${value}`;
  }

  return value;
};

export const toCsvHeader = (key: string): string =>
  CSV_HEADER_OVERRIDES[key] ??
  key.replace(/[A-Z]/g, (char) => `_${char.toLowerCase()}`);

// Column order agreed with the consumers of the export, expressed in API field
// names so that renaming a header above cannot silently reorder the file.
// Fields that are not listed are appended in the order the API returns them,
// so new backend fields show up without touching this file.
const CSV_COLUMN_ORDER = [
  'queryId',
  'queryExecutionDurationMs',
  'dbInstanceAddress',
  'clientAddress',
  'databaseName',
  'serviceName',
  'username',
  'collection',
  'operation',
  'planSummary',
  'clientAppName',
  'operationStartTime',
  'queryCollectTime',
  'queryRawJson',
].map(toCsvHeader);

const columnRank = (column: string): number => {
  const index = CSV_COLUMN_ORDER.indexOf(column);

  return index === -1 ? CSV_COLUMN_ORDER.length : index;
};

// export-to-csv rejects object values outright, so anything that is not a
// primitive has to be flattened or serialized before it reaches the generator.
export const toCsvValue = (value: unknown): CsvValue => {
  if (value === null || value === undefined) {
    return '';
  }

  if (typeof value === 'string') {
    return sanitizeCsvCell(value);
  }

  if (typeof value === 'number' || typeof value === 'boolean') {
    return value;
  }

  return sanitizeCsvCell(JSON.stringify(value) ?? '');
};

// Recurses so that nested payloads are hoisted into the parent key space,
// which is what lets a new database-specific payload export without any
// mapping.
const flattenToCsvRow = (source: Record<string, unknown>): CsvRow => {
  const row: CsvRow = {};

  for (const [key, value] of Object.entries(source)) {
    if (isPlainObject(value)) {
      Object.assign(row, flattenToCsvRow(value));
    } else {
      row[toCsvHeader(key)] = toCsvValue(value);
    }
  }

  return row;
};

export const mapQueryToCsvRow = (query: QueryData): CsvRow =>
  flattenToCsvRow(query as unknown as Record<string, unknown>);

// Columns are collected across every row: export-to-csv derives headers from
// the first row only, which would drop fields that are unset on it.
export const collectCsvColumns = (rows: CsvRow[]): string[] => {
  const columns = [...new Set(rows.flatMap((row) => Object.keys(row)))];

  // Sorting is stable, so unlisted columns share a rank and keep the order the
  // API returned them in.
  return columns.sort((a, b) => columnRank(a) - columnRank(b));
};

export const buildRtaExportFilename = (date = new Date()): string => {
  const timestamp = format(date, 'yyyyMMdd_HHmmss');

  return `mongodb_rta_export_${timestamp}`;
};

export const exportRtaQueriesToCsv = (queries: QueryData[]): void => {
  if (queries.length === 0) {
    return;
  }

  const rows = queries.map(mapQueryToCsvRow);
  const csvConfig = mkConfig({
    columnHeaders: collectCsvColumns(rows),
    filename: buildRtaExportFilename(),
  });
  const csv = generateCsv(csvConfig)(rows);
  download(csvConfig)(csv);
};
