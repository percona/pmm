import { format } from 'date-fns';
import { download, generateCsv, mkConfig } from 'export-to-csv';
import { ADVISOR_RESULT_STATUS, SEVERITY } from 'lib/constants';
import type { AdvisorRun, Insight } from 'types/advisors.types';
import { Severity } from 'types/severity.types';
import { type CsvRow, toCsvValue } from 'utils/csv.utils';
import { TRIGGERED_BY_LABEL } from '../insights/AdvisorInsights.utils';

// every insight field, in the order the columns appear; headers are the
// snake_case names the API uses, as in the RTA export
const CSV_FIELDS: (keyof Insight)[] = [
  'checkedAt',
  'checkName',
  'summary',
  'severity',
  'status',
  'category',
  'serviceName',
  'serviceType',
  'serviceId',
  'nodeName',
  'nodeId',
  'environment',
  'cluster',
  'replicationSet',
  'region',
  'az',
  'description',
  'outcome',
  'readMoreUrl',
  'labels',
  'isRead',
  'triggeredBy',
  'runId',
  'id',
];

const toCsvHeader = (field: string): string =>
  field.replace(/[A-Z]/g, (char) => `_${char.toLowerCase()}`);

export const mapInsightToCsvRow = (insight: Insight): CsvRow => {
  // enums read as they do on the Insights page
  const values: Partial<Record<keyof Insight, unknown>> = {
    ...insight,
    // a pending or not run check has no severity
    severity:
      insight.severity === Severity.unspecified
        ? ''
        : SEVERITY[insight.severity],
    status: ADVISOR_RESULT_STATUS[insight.status],
    triggeredBy: TRIGGERED_BY_LABEL[insight.triggeredBy],
  };

  return Object.fromEntries(
    CSV_FIELDS.map((field) => [toCsvHeader(field), toCsvValue(values[field])])
  );
};

export const buildRunInsightsFilename = (
  run: AdvisorRun,
  failedOnly = false
): string =>
  `advisor_run_${failedOnly ? 'failed_' : ''}insights_${format(new Date(run.startedAt), 'yyyyMMdd_HHmmss')}`;

export const exportRunInsightsToCsv = (
  run: AdvisorRun,
  insights: Insight[],
  failedOnly = false
): void => {
  const csvConfig = mkConfig({
    columnHeaders: CSV_FIELDS.map(toCsvHeader),
    filename: buildRunInsightsFilename(run, failedOnly),
  });
  const csv = generateCsv(csvConfig)(insights.map(mapInsightToCsvRow));
  download(csvConfig)(csv);
};
