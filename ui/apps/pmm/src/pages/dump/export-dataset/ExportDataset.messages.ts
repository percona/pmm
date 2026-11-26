export const Messages = {
  pageTitle: 'Create dataset',
  sections: {
    services: 'Services',
    timeRange: 'Time range',
    data: 'Data to include',
    security: 'Security',
    serverLoad: 'Server load',
  },
  timeRangeDescription:
    'A longer window makes a larger archive and a longer export.',
  allServices: (count?: number) =>
    count === undefined ? 'All services' : `All services (${count})`,
  selectedServices: 'Only selected services',
  services: 'Services',
  noServices: 'No services available',
  servicesError: 'Could not load services.',
  retry: 'Retry',
  startTime: 'Start time',
  endTime: 'End time',
  exportQan: 'Include Query Analytics (QAN)',
  exportQanHelp:
    'Adds query examples, which can contain data from your queries.',
  enableEncryption: 'Encrypt the archive',
  enableEncryptionHelp: 'You need the password to open the archive.',
  encryptionPassword: 'Encryption password',
  passwordRules:
    'At least 8 characters, with a letter, a number and a special character.',
  passwordWarning:
    "PMM doesn't store this password and can't recover it. Share it with Percona Support in your ticket, separately from the archive.",
  ignoreLoad: 'Export even when PMM Server is busy',
  ignoreLoadHelp:
    "When off, the export pauses while PMM Server's CPU or memory is high and stops if the load becomes critical. When on, it runs regardless, and dashboards can slow down until it finishes.",
  create: 'Create dataset',
  creating: 'Creating...',
  cancel: 'Cancel',
  success: 'Dataset creation started. It runs in the background.',
  validation: {
    validRange: 'Please select a valid time range.',
    futureDate: 'Dates cannot be in the future.',
    servicesRequired: 'Select at least one service.',
  },
};
