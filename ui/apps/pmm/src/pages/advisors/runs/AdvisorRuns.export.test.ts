import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  AdvisorCheckResultStatus,
  AdvisorCheckTriggeredBy,
  AdvisorInterval,
  type AdvisorRun,
  AdvisorRunStatus,
  type Insight,
} from 'types/advisors.types';
import { Severity } from 'types/severity.types';
import {
  buildRunInsightsFilename,
  exportRunInsightsToCsv,
  mapInsightToCsvRow,
} from './AdvisorRuns.export';

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

const RUN: AdvisorRun = {
  id: 'run-1',
  triggeredBy: AdvisorCheckTriggeredBy.user,
  status: AdvisorRunStatus.completed,
  startedAt: '2026-08-04T19:57:28',
  finishedAt: '2026-08-04T19:59:35',
  plannedChecksCount: 1,
  plannedServicesCount: 1,
  checksCount: 1,
  servicesCount: 1,
  findingsCount: 1,
  errorsCount: 0,
  severityCounts: [{ severity: Severity.warning, count: 1 }],
  checkNames: [],
  serviceIds: [],
  intervals: [],
};

const INSIGHT: Insight = {
  id: 'result-1',
  checkName: 'mysql_version_check',
  category: 'Configuration',
  interval: AdvisorInterval.standard,
  serviceId: 'service-1',
  serviceName: 'mysql-prod',
  serviceType: 'mysql',
  nodeId: 'node-1',
  nodeName: 'node-1',
  status: AdvisorCheckResultStatus.failed,
  summary: 'MySQL is outdated',
  description: 'Newer version of MySQL is available',
  readMoreUrl: 'https://percona.com',
  severity: Severity.warning,
  labels: { env: 'prod' },
  checkedAt: '2026-08-04T19:58:00Z',
  isRead: false,
  runId: 'run-1',
  triggeredBy: AdvisorCheckTriggeredBy.user,
  outcome: '=HYPERLINK("x")',
  environment: 'prod',
  cluster: 'mysql-cluster',
  replicationSet: 'rs1',
  region: '',
  az: '',
};

describe('exportRunInsightsToCsv', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('maps every insight field to a column, with enums as on the Insights page', () => {
    expect(mapInsightToCsvRow(INSIGHT)).toEqual({
      checked_at: '2026-08-04T19:58:00Z',
      check_name: 'mysql_version_check',
      summary: 'MySQL is outdated',
      severity: 'Warning',
      status: 'Failed',
      category: 'Configuration',
      interval: 'Standard',
      service_name: 'mysql-prod',
      service_type: 'mysql',
      service_id: 'service-1',
      node_name: 'node-1',
      node_id: 'node-1',
      environment: 'prod',
      cluster: 'mysql-cluster',
      replication_set: 'rs1',
      region: '',
      az: '',
      description: 'Newer version of MySQL is available',
      // check output is untrusted, so it cannot start a formula
      outcome: '\'=HYPERLINK("x")',
      read_more_url: 'https://percona.com',
      labels: '{"env":"prod"}',
      is_read: false,
      triggered_by: 'User',
      run_id: 'run-1',
      id: 'result-1',
    });
  });

  it('leaves the severity and check time of a pending check empty', () => {
    const row = mapInsightToCsvRow({
      ...INSIGHT,
      status: AdvisorCheckResultStatus.pending,
      severity: Severity.unspecified,
      checkedAt: null,
    });

    expect(row.status).toBe('Pending');
    expect(row.severity).toBe('');
    expect(row.checked_at).toBe('');
  });

  it("names the file after the run's start time", () => {
    expect(buildRunInsightsFilename(RUN)).toBe(
      'advisor_run_insights_20260804_195728'
    );
    expect(buildRunInsightsFilename(RUN, true)).toBe(
      'advisor_run_failed_insights_20260804_195728'
    );
  });

  it('downloads every insight with the full header row', () => {
    exportRunInsightsToCsv(RUN, [INSIGHT, { ...INSIGHT, id: 'result-2' }]);

    expect(mkConfig).toHaveBeenCalledWith({
      columnHeaders: Object.keys(mapInsightToCsvRow(INSIGHT)),
      filename: 'advisor_run_insights_20260804_195728',
    });
    const generate = vi.mocked(generateCsv).mock.results[0].value;
    expect(generate).toHaveBeenCalledWith([
      mapInsightToCsvRow(INSIGHT),
      mapInsightToCsvRow({ ...INSIGHT, id: 'result-2' }),
    ]);
    const save = vi.mocked(download).mock.results[0].value;
    expect(save).toHaveBeenCalledWith('csv-content');
  });
});
