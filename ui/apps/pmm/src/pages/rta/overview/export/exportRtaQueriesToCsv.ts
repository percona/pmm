import { format, formatDuration } from 'date-fns';
import { download, generateCsv, mkConfig } from 'export-to-csv';
import { QueryData } from 'types/rta.types';
import { Messages } from '../table/OverviewTable.messages';

const CSV_FORMULA_PREFIX = /^[=+\-@\t\r]/;

export const sanitizeCsvCell = (value: string): string => {
  if (CSV_FORMULA_PREFIX.test(value)) {
    return `'${value}`;
  }

  return value;
};

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
  [Messages.columns.queryText]: sanitizeCsvCell(query.queryText),
  [Messages.columns.host]: sanitizeCsvCell(query.serviceName),
  [Messages.columns.operationId]: sanitizeCsvCell(query.queryId),
  [Messages.columns.elapsedTime]: sanitizeCsvCell(
    formatElapsedTimeForExport(query.queryExecutionDurationMs)
  ),
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
