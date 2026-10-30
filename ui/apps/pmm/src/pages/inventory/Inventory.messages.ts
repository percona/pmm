import { ServiceStatus } from 'types/services.types';

export const Messages = {
  title: 'Inventory',
  tabs: {
    services: 'Services',
    nodes: 'Nodes',
  },
  status: {
    [ServiceStatus.up]: 'Up',
    [ServiceStatus.down]: 'Down',
    [ServiceStatus.unknown]: 'Unknown',
    [ServiceStatus.unspecified]: 'N/A',
  },
  statusInfo: {
    [ServiceStatus.up]: 'Database is running.',
    [ServiceStatus.down]: 'Database is not running.',
    [ServiceStatus.unknown]:
      'Indicates agent unavailability or incorrect database status. Troubleshoot the configuration.',
    [ServiceStatus.unspecified]:
      'Arises when external services are monitored and required metrics cannot be checked to determine service availability.',
  },
  agentsSummary: (running: number, total: number) =>
    running === 0
      ? `${total}/${total} not running`
      : `${running}/${total} running`,
  columns: {
    status: 'Status',
    monitoring: 'Monitoring',
    address: 'Address',
  },
  details: {
    agents: 'Agents',
    labels: 'Labels',
  },
  actions: {
    delete: 'Delete',
    rowActions: 'Row actions',
  },
  deleteModal: {
    cancel: 'Cancel',
    forceMode: 'Force mode',
  },
  selected: (count: number) => `${count} selected`,
};
