export const Messages = {
  title: 'Your PMM instance is up-to-date. ',
  seeReleaseNotes: 'See Release Notes',
  notify: " We'll notify you when a new update becomes available.",
  dot: '.',
  pmmUpdate: (version?: string) => `PMM ${version} update`,
  inProgress: 'In progress',
  step: 'Step 2 of 2: Update PMM Client instances',
  stepDescription:
    "To complete this update, it's crucial to also update all of your PMM Client instances. Before proceeding with the Client update process, check the instances that require updates in the list below and review the provided instructions.",
  howToUpdate: 'How to update PMM Client',
  refreshList: 'Refresh list',
  table: {
    node: 'Node',
    client: 'PMM Client (UUID)',
    version: 'Version',
    severity: 'Status',
    empty: 'No clients matching the filter',
  },
  filter: {
    label: 'Filter',
    all: 'All versions',
    update: 'Update required',
    critical: 'Critical to update',
  },
  severity: {
    critical: 'Critical to update',
    required: 'Update required',
    upToDate: 'Up-to-date',
    unsupported: 'Unsupported',
    unspecified: 'Unspecified',
  },
};
