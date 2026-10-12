import { NodeRole } from 'types/ha.types';

export const Messages = {
  columns: {
    nodeId: 'Node ID',
    nodeName: 'Node Name',
    nodeType: 'Node Type',
    services: 'Services',
    isPmmServerNode: 'PMM Server node',
  },
  pmmServerOnly: 'List PMM Server only nodes',
  noServices: 'No services',
  servicesCount: (count: number) => `${count} services`,
  details: {
    nodeId: 'Node ID',
    serviceNames: 'Service Names',
    properties: 'Attributes',
  },
  haRole: {
    [NodeRole.leader]: 'Leader',
    [NodeRole.follower]: 'Follower',
    [NodeRole.unspecified]: 'Unspecified',
  },
  empty: 'No nodes available',
  delete: {
    title: 'Confirm action',
    message: (count: number) =>
      `Are you sure that you want to permanently delete ${count} node${count > 1 ? 's' : ''}?`,
    forceDescription:
      'Force mode is going to delete all agents and services associated with the nodes',
    confirm: 'Proceed',
    success: (deleted: number, total: number) =>
      `${deleted} of ${total} nodes successfully deleted`,
  },
};
