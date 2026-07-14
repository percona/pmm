export const Messages = {
  title: 'Advisor checks',
  description:
    'Run automated checks against your connected databases to identify potential security threats, performance degradation, data loss and data corruption.',
  noData: 'No advisor checks found',
  columns: {
    check: 'Check',
    description: 'Description',
    category: 'Category',
    vendor: 'Vendor',
    interval: 'Interval',
    status: 'Status',
  },
  status: {
    enabled: 'Enabled',
    disabled: 'Disabled',
  },
  searchPlaceholder: 'Search checks',
  filters: {
    all: 'All',
    category: 'Category',
    vendor: 'Vendor',
    interval: 'Interval',
    status: 'Status',
    clear: 'Clear filters',
  },
  runAll: 'Run all',
  runSelected: 'Run selected',
  addAdvisor: 'Add advisor',
  viewResults: 'View results',
  run: 'Run',
  success: {
    checksStarted:
      'Advisor checks started, batch ID copied to clipboard',
    checkStarted: (summary: string) =>
      `Check "${summary}" started, batch ID copied to clipboard`,
    checkEnabled: (summary: string) => `Check "${summary}" enabled`,
    checkDisabled: (summary: string) => `Check "${summary}" disabled`,
    intervalChanged: (summary: string, interval: string) =>
      `Check "${summary}" interval changed to "${interval}"`,
  },
};
