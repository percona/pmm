const plural = (count: number, one: string, many: string) =>
  `${count} ${count === 1 ? one : many}`;

export const Messages = {
  title: 'Advisor insights',
  description:
    'Find and fix the issues that Advisor checks report on your services, then re-run a check to confirm the fix.',
  noData:
    'No insights match. Clear the filters, or run checks from the Catalog.',
  readMoreAvailable: 'Double-click the row for Read more',
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
    pendingOutcome: "The run hasn't reached this check yet.",
  },
  viewResults: 'View results',
  filters: {
    all: 'All',
    runId: 'Run ID',
    clear: 'Clear filters',
    refresh: 'Refresh',
    refreshTooltip: 'Reload results and filters',
    email: 'Email insights',
    emailTooltip: 'Email the filtered insights',
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
      `Emailed ${plural(insights, 'insight', 'insights')} to ${plural(recipients, 'recipient', 'recipients')}`,
  },
  success: {
    rerunStarted: (check: string, service: string) =>
      `Check "${check}" started on ${service}, run ID copied to clipboard`,
    checkEnabled: (check: string, service: string) =>
      `Check "${check}" enabled for ${service}`,
    checkDisabled: (check: string, service: string) =>
      `Check "${check}" disabled for ${service}`,
    copied: 'Insight copied to clipboard',
    urlCopied: 'Insight URL copied to clipboard',
  },
};
