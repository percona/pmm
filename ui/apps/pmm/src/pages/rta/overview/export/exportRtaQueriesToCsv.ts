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
  queryExecutionDurationMs?: number | null
): number | '' => {
  if (queryExecutionDurationMs == null) {
    return '';
  }

  return queryExecutionDurationMs;
};

export const mapQueryToCsvRow = (query: QueryData) => {
  const { mongoDbPayload } = query;

  return {
    operation_id: sanitizeCsvCell(query.queryId),
    elapsed_exec_time_sec: formatElapsedExecTimeSec(
      query.queryExecutionDurationMs
    ),
    db_instance_address: sanitizeCsvCell(mongoDbPayload.dbInstanceAddress),
    client_address: sanitizeCsvCell(query.clientAddress),
    database_name: sanitizeCsvCell(mongoDbPayload.databaseName),
    service: sanitizeCsvCell(query.serviceName),
    user_name: sanitizeCsvCell(mongoDbPayload.username),
    collection: sanitizeCsvCell(mongoDbPayload.collection ?? ''),
    operation: sanitizeCsvCell(mongoDbPayload.operation),
    plan_summary: sanitizeCsvCell(mongoDbPayload.planSummary),
    client_app_name: sanitizeCsvCell(mongoDbPayload.clientAppName),
    operation_start_time: sanitizeCsvCell(mongoDbPayload.operationStartTime),
    data_capture_time: sanitizeCsvCell(query.queryCollectTime),
    raw_query: sanitizeCsvCell(query.queryRawJson),
  };
};

export const buildRtaExportFilename = (date = new Date()): string => {
  const timestamp = format(date, 'yyyyMMdd_HHmmss');

  return `mongodb_rta_export_${timestamp}`;
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
