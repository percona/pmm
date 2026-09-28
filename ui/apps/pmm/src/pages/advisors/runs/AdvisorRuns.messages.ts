export const Messages = {
  title: 'Advisor runs',
  description:
    'Every Advisor checks execution, most recent first. Open a run to see the insights it produced.',
  noData: 'No Advisor runs recorded yet',
  running: 'Running…',
  runAllChecks: 'Run all checks',
  columns: {
    startedAt: 'Started',
    duration: 'Duration',
    triggeredBy: 'Triggered by',
    findings: 'Findings',
    severity: 'Severity',
    errors: 'Failed',
    checks: 'Checks',
    services: 'Services',
    actions: 'Actions',
  },
  tooltips: {
    errors:
      'Checks that could not be executed, as opposed to checks that found an issue',
    findings: 'Checks that detected an issue',
  },
  actions: {
    more: 'More actions',
    viewInsights: 'View insights',
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
