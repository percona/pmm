export const Messages = {
  title: 'PMM Dump',
  description:
    'Export metrics and Query Analytics data from this PMM Server into an archive you can download or upload to Percona Support.',
  learnMore: 'Learn more',
  createDataset: 'Create dataset',
  creatingBlocked:
    'Another dataset is being created. A new one starts only after that export finishes.',
  loadError: 'Could not load datasets.',
  notAvailable: 'N/A',
  emptyTitle: 'No datasets yet',
  emptyDescription:
    "A dataset is an archive of this server's metrics, and optionally Query Analytics data, for a time window you choose. Percona Support uses it to investigate issues.",
  allServices: 'All services',
  moreServices: (count: number) => `+${count}`,
  encrypted: 'Encrypted',
  encryptedHint:
    'Opening the archive needs the password set when the dataset was created.',
  columns: {
    id: 'ID',
    status: 'Status',
    services: 'Services',
    dataFrom: 'Data from',
    created: 'Created',
  },
  status: {
    ready: 'Ready',
    creating: 'Creating…',
    failed: 'Failed',
    unknown: 'Unknown',
    startedAgo: (time: string) => `started ${time}`,
    seeWhatWentWrong: 'See what went wrong',
  },
  actions: {
    download: 'Download',
    sendToSupport: 'Send to Support',
    viewLogs: 'View logs',
    delete: 'Delete',
  },
  reasons: {
    notReady: 'Available once the dataset is ready',
    bulkNotReady: (count: number, total: number) =>
      `${count} of ${total} selected datasets ${count === 1 ? "isn't" : "aren't"} ready`,
  },
  downloading: (count: number) =>
    `Downloading ${count} archives. If your browser asks, allow multiple downloads.`,
  delete: {
    title: (count: number) =>
      count === 1 ? 'Delete this dataset?' : `Delete ${count} datasets?`,
    message:
      "The archives are removed from PMM Server. Copies you already downloaded aren't affected.",
    confirm: (count: number) =>
      count === 1 ? 'Delete dataset' : `Delete ${count} datasets`,
    creatingWarning:
      "Deleting a dataset that is still being created doesn't stop its export.",
    success: 'Dataset deleted successfully.',
    multipleSuccess: 'Datasets deleted successfully.',
  },
  logs: {
    title: 'Dataset logs',
    running: "Export running. New lines appear as they're written.",
    finished: 'Export finished.',
    empty: 'No logs available.',
    loading: 'Loading logs',
    error: 'Could not load dataset logs.',
    close: 'Close',
  },
};
