import { format } from 'date-fns';
import { download, generateCsv, mkConfig } from 'export-to-csv';
import { QueryData } from 'types/rta.types';

const CSV_FORMULA_PREFIX = /^[=+\-@\t\r]/;

export const sanitizeCsvCell = (value: string): string => {
  if (CSV_FORMULA_PREFIX.test(value)) {
    return `'${value}`;
  }

  return value;
};

export const formatElapsedExecTimeSec = (
  queryExecutionDurationSec?: number | null
): number | '' => {
  if (
    queryExecutionDurationSec === null ||
    queryExecutionDurationSec === undefined
  ) {
    return '';
  }

  return queryExecutionDurationSec;
};

export const mapQueryToCsvRow = (query: QueryData) => {
  const { mongoDbPayload, mySqlPayload } = query;
  // Fields common to all database types come from whichever payload is present.
  const payload = mongoDbPayload ?? mySqlPayload;

  return {
    operation_id: sanitizeCsvCell(query.queryId),
    elapsed_exec_time_sec: formatElapsedExecTimeSec(
      // QueryData stores seconds here despite the Ms suffix in the field name.
      query.queryExecutionDurationMs
    ),
    db_instance_address: sanitizeCsvCell(payload?.dbInstanceAddress ?? ''),
    client_address: sanitizeCsvCell(query.clientAddress),
    database_name: sanitizeCsvCell(payload?.databaseName ?? ''),
    service: sanitizeCsvCell(query.serviceName),
    user_name: sanitizeCsvCell(payload?.username ?? ''),
    // MongoDB-specific columns; empty for other database types.
    collection: sanitizeCsvCell(mongoDbPayload?.collection ?? ''),
    operation: sanitizeCsvCell(mongoDbPayload?.operation ?? ''),
    plan_summary: sanitizeCsvCell(mongoDbPayload?.planSummary ?? ''),
    client_app_name: sanitizeCsvCell(mongoDbPayload?.clientAppName ?? ''),
    operation_start_time: sanitizeCsvCell(
      mongoDbPayload?.operationStartTime ?? ''
    ),
    // MySQL-specific columns; empty for other database types.
    command: sanitizeCsvCell(mySqlPayload?.command ?? ''),
    state: sanitizeCsvCell(mySqlPayload?.state ?? ''),
    program_name: sanitizeCsvCell(mySqlPayload?.programName ?? ''),
    rows_examined: sanitizeCsvCell(String(mySqlPayload?.rowsExamined ?? '')),
    rows_sent: sanitizeCsvCell(String(mySqlPayload?.rowsSent ?? '')),
    full_scan: mySqlPayload ? (mySqlPayload.fullScan ? 'yes' : 'no') : '',
    data_capture_time: sanitizeCsvCell(query.queryCollectTime),
    raw_query: sanitizeCsvCell(query.queryRawJson),
  };
};

export const buildRtaExportFilename = (date = new Date()): string => {
  const timestamp = format(date, 'yyyyMMdd_HHmmss');

  return `rta_export_${timestamp}`;
};

export const exportRtaQueriesToCsv = (queries: QueryData[]): void => {
  if (queries.length === 0) {
    return;
  }

  const csvConfig = mkConfig({
    useKeysAsHeaders: true,
    filename: buildRtaExportFilename(),
  });
  const csv = generateCsv(csvConfig)(queries.map(mapQueryToCsvRow));
  download(csvConfig)(csv);
};
