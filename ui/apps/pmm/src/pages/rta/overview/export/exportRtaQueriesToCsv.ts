import { format, formatDuration } from 'date-fns';
import { download, generateCsv, mkConfig } from 'export-to-csv';
import { QueryData } from 'types/rta.types';

export const formatElapsedTimeForExport = (
  queryExecutionDurationMs?: number | null
): string => {
  if (queryExecutionDurationMs == null) {
    return '';
  }

  return formatDuration(
    {
      seconds: queryExecutionDurationMs,
    },
    {
      format: ['seconds'],
    }
  );
};

export const mapQueryToCsvRow = (query: QueryData) => ({
  'Operation ID': query.queryId,
  Service: query.serviceName,
  'Query Text': query.queryText,
  'Elapsed Time': formatElapsedTimeForExport(query.queryExecutionDurationMs),
  'Plan Summary': query.mongoDbPayload.planSummary ?? '',
});

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
