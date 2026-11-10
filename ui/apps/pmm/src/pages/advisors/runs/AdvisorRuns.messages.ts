export const Messages = {
  title: 'Advisor run history',
  description:
    'Check that your Advisor runs started on time, finished and reached all your services. Open a run to review what it found, or run it again after you fix what it missed.',
  noData:
    'No runs yet. Run checks from the Catalog, or wait for the next scheduled run.',
  running: 'Running…',
  queued: 'Queued…',
  interrupted: 'Interrupted',
  aborted: 'Aborted',
  runAllChecks: 'Run all checks',
  columns: {
    startedAt: 'Started',
    duration: 'Duration',
    triggeredBy: 'Triggered by',
    findings: 'Findings',
    severity: 'Severity',
    errors: 'Errors',
    checks: 'Checks',
    services: 'Services',
    actions: 'Actions',
  },
  tooltips: {
    checks:
      'Checks that passed or found an issue on at least one service, out of the checks the run planned. Checks that could not run are counted in Errors.',
    services:
      'Services on which at least one check passed or found an issue, out of the services the run planned to cover',
    nothingToCheck:
      'There was nothing to check: no MySQL, PostgreSQL or MongoDB services, or every check that applies to them is disabled',
    errors:
      "Checks that couldn't run. Checks that found an issue are counted under Findings.",
    findings: 'Checks that found an issue',
    queued: 'Waiting to start. This takes up to a minute.',
    interrupted:
      'PMM restarted, or the HA leader changed, before the run finished. The insights it saved are kept. Any checks it didn\'t reach show as "Not run".',
    aborted:
      'The run stopped before any check ran. To find out why, look in the PMM Server logs.',
    waited: (scheduled: string, waited: string, started: string, who: string) =>
      `Scheduled for ${scheduled} UTC. Started ${waited} late because it waited for ${who}, which started at ${started} UTC.`,
    userRun: 'a user run',
    scheduledRun: 'a scheduled run',
    startedLate: 'Started late',
  },
  actions: {
    more: 'More actions',
    viewInsights: 'View insights',
    exportAll: 'Export all checks',
    exportFailed: 'Export failed checks',
    runAgain: 'Run again',
    copyRunId: 'Copy run ID',
  },
  filters: {
    triggeredBy: 'Triggered by',
    schedule: 'Schedule',
    deletedSchedule: 'Deleted schedule',
    runId: 'Run ID',
    all: 'All',
    clear: 'Clear filters',
    refresh: 'Refresh',
    refreshTooltip: 'Reload the list of runs',
  },
  success: {
    runIdCopied: 'Run ID copied to clipboard',
    checksStarted: 'Advisor checks started',
  },
  info: {
    noInsightsToExport: 'This run has no insights to export',
    noFailedInsightsToExport: 'This run has no failed insights to export',
  },
};
