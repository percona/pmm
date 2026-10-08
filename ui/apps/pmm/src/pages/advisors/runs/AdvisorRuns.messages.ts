export const Messages = {
  title: 'Advisor runs',
  description:
    'Every Advisor checks execution, most recent first. Open a run to see the insights it produced.',
  noData: 'No Advisor runs recorded yet',
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
      'Checks that could not be executed, as opposed to checks that found an issue',
    findings: 'Checks that detected an issue',
    queued:
      'Requested and waiting for PMM to start it, which takes up to a minute',
    interrupted:
      'PMM restarted, or the HA leader changed, before the run finished. The insights it saved are kept; the checks it did not reach show as Not run.',
    aborted:
      'The run stopped before running any check. See the PMM Server logs for the cause.',
  },
  actions: {
    more: 'More actions',
    viewInsights: 'View insights',
    runAgain: 'Run again',
    copyRunId: 'Copy run ID',
  },
  filters: {
    triggeredBy: 'Triggered by',
    all: 'All',
    clear: 'Clear filters',
    refresh: 'Refresh',
    refreshTooltip: 'Reload the list of runs',
  },
  success: {
    runIdCopied: 'Run ID copied to clipboard',
    checksStarted: 'Advisor checks started',
  },
};
