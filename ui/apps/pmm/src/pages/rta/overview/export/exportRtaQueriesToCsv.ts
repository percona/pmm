import type { QueryData } from 'types/rta.types';
import { ServiceType } from 'types/services.types';
import { type CsvRow, downloadCsv, toCsvValue } from 'utils/csv.utils';
import { isPlainObject } from 'utils/object.utils';
import {
  lockTimeMs,
  soleBlocker,
  sqlPayload,
} from '../table/OverviewTable.utils';

// MongoDB keeps the mongodb_rta_export name it shipped with in 3.9.
const RTA_EXPORT_FILENAME_PREFIX: Partial<Record<ServiceType, string>> = {
  [ServiceType.mongodb]: 'mongodb_rta_export',
  [ServiceType.mysql]: 'mysql_rta_export',
  [ServiceType.posgresql]: 'postgresql_rta_export',
};

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
  'command',
  'state',
  'programName',
  'rowsExamined',
  'rowsSent',
  'fullScan',
  'lockTimeMs',
  'blockedStatus',
  'lockedTable',
  'lockedIndex',
  'blockingConnId',
  'blockingQuery',
  'blockingQueryTruncated',
  'queryTextTruncated',
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

export const mapQueryToCsvRow = (query: QueryData): CsvRow => {
  const row = flattenToCsvRow(query as unknown as Record<string, unknown>);

  // Serialized, it would be one unfilterable JSON cell; only a sole blocker is written below,
  // and blocked_status still finds a statement held up by several.
  delete row[toCsvHeader('blockedBy')];

  // A spreadsheet can sum milliseconds; it cannot sum "0.000003s".
  delete row[toCsvHeader('lockTime')];
  const lockTime = lockTimeMs(query.mySqlPayload?.lockTime);
  if (lockTime !== undefined) {
    row[toCsvHeader('lockTimeMs')] = lockTime;
  }

  // The API omits a false flag, and an empty cell would not say whether the text is complete.
  const payload = sqlPayload(query);
  if (payload) {
    row[toCsvHeader('queryTextTruncated')] = Boolean(
      payload.queryTextTruncated
    );
  }

  // Flattening would let PostgreSQL's own query_id overwrite operation_id, the backend pid.
  if (query.postgresqlPayload) {
    row[toCsvHeader('queryId')] = toCsvValue(query.queryId);
    row[toCsvHeader('pgQueryId')] = toCsvValue(query.postgresqlPayload.queryId);
  }

  const blocker = soleBlocker(query);
  if (blocker) {
    row[toCsvHeader('blockingConnId')] = toCsvValue(blocker.blockingConnId);
    row[toCsvHeader('blockingQuery')] = toCsvValue(blocker.blockingQuery);
    // Written out like query_text_truncated, for the same reason.
    row[toCsvHeader('blockingQueryTruncated')] = Boolean(
      blocker.blockingQueryTruncated
    );
  }

  return row;
};

const rtaExportFilenamePrefix = (serviceType?: ServiceType): string =>
  (serviceType && RTA_EXPORT_FILENAME_PREFIX[serviceType]) ?? 'rta_export';

export const exportRtaQueriesToCsv = (
  queries: QueryData[],
  serviceType?: ServiceType
): void =>
  downloadCsv(
    queries.map(mapQueryToCsvRow),
    rtaExportFilenamePrefix(serviceType),
    RTA_CSV_COLUMNS
  );
