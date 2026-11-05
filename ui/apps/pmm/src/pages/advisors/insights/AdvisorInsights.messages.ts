const plural = (count: number, one: string, many: string) =>
  `${count} ${count === 1 ? one : many}`;

export const Messages = {
  title: 'Advisor insights',
  description:
    'Results of the advisor checks that ran against your monitored services.',
  noData: 'No advisor insights found',
  readMoreAvailable: 'Documentation available in the details pane',
  actions: {
    more: 'More actions',
    viewDetails: 'View details',
    rerunNow: 'Re-run now',
    filterByRunId: 'Filter by run ID',
    filterByCheckName: 'Filter by name',
    copyAsText: 'Copy as text',
    copyUrl: 'Copy URL',
    disableCheck: 'Disable check for service',
    enableCheck: 'Enable check for service',
  },
  details: {
    title: 'Advisor insight details',
    maximize: 'Maximize',
    minimize: 'Minimize',
    close: 'Close',
    summary: 'Summary',
    severity: 'Severity',
    status: 'Status',
    triggeredBy: 'Triggered by',
    description: 'Description',
    outcome: 'Outcome',
    readMore: 'Read more',
    service: 'Service',
    technology: 'Technology',
    node: 'Node',
    category: 'Category',
    environment: 'Environment',
    cluster: 'Cluster',
    replicationSet: 'Replication set',
    region: 'Region',
    checkedAt: 'Checked at',
    checkId: 'Check ID',
    checkName: 'Check name',
    runId: 'Run ID',
    labels: 'Labels',
    pendingOutcome: 'The check is waiting for its run to execute it.',
  },
  viewResults: 'View results',
  filters: {
    all: 'All',
    runId: 'Run ID',
    clear: 'Clear filters',
    refresh: 'Refresh',
    refreshTooltip:
      'Reload the check results and re-populate the filter options',
    email: 'Email insights',
    emailTooltip: 'Email the insights matching the filters',
    service: 'Service',
    node: 'Node',
    category: 'Category',
    severity: 'Severity',
    status: 'Status',
  },
  columns: {
    summary: 'Summary',
    service: 'Service',
    category: 'Category',
    severity: 'Severity',
    status: 'Status',
    checkedAt: 'Checked at',
    actions: 'Actions',
  },
  email: {
    title: 'Email insights',
    filtered: (count: number) =>
      `Email ${plural(count, 'insight', 'insights')} matching the current filters.`,
    unfiltered: (count: number) =>
      `No filters are set, so this emails all ${plural(count, 'insight', 'insights')}.`,
    tooMany: (count: number, max: number) =>
      `${count} insights match, more than the ${max} one email can carry. Narrow the filters to ${max} insights or fewer.`,
    required: 'Enter at least one address.',
    cancel: 'Cancel',
    send: 'Send',
    sent: (insights: number, recipients: number) =>
      `Emailed ${plural(insights, 'insight', 'insights')} to ${plural(recipients, 'recipient', 'recipients')}.`,
  },
  success: {
    rerunStarted: (check: string, service: string) =>
      `Check "${check}" started on ${service}, run ID copied to clipboard.`,
    checkEnabled: (check: string, service: string) =>
      `Check "${check}" enabled for ${service}`,
    checkDisabled: (check: string, service: string) =>
      `Check "${check}" disabled for ${service}`,
    copied: 'Insight copied to clipboard',
    urlCopied: 'Insight URL copied to clipboard',
  },
};
