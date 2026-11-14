import type { QueryData } from 'types/rta.types';
import { type CsvRow, downloadCsv, toCsvValue } from 'utils/csv.utils';
import { isPlainObject } from 'utils/object.utils';

const RTA_EXPORT_FILENAME_PREFIX = 'mongodb_rta_export';

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

export const toCsvHeader = (key: string): string =>
  CSV_HEADER_OVERRIDES[key] ??
  key.replace(/[A-Z]/g, (char) => `_${char.toLowerCase()}`);

// Column order agreed with the consumers of the export, expressed in API field
// names so that renaming a header above cannot silently reorder the file.
// Fields that are not listed are appended in the order the API returns them,
// so new backend fields show up without touching this file.
export const RTA_CSV_COLUMNS = [
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

export const exportRtaQueriesToCsv = (queries: QueryData[]): void =>
  downloadCsv(
    queries.map(mapQueryToCsvRow),
    RTA_EXPORT_FILENAME_PREFIX,
    RTA_CSV_COLUMNS
  );
