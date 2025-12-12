import {
  GetHANodesResponse,
  GetHAStatusResponse,
  NodeRole,
} from 'types/ha.types';

export const HA_STATUS_MOCK: GetHAStatusResponse = {
  status: 'Enabled',
};

export const HA_NODES_MOCK_HEALTHY: GetHANodesResponse = {
  nodes: [
    {
      nodeName: 'pmm-ha-0',
      role: NodeRole.follower,
      status: 'alive',
    },
    {
      nodeName: 'pmm-ha-1',
      role: NodeRole.leader,
      status: 'alive',
    },
    {
      nodeName: 'pmm-ha-2',
      role: NodeRole.follower,
      status: 'alive',
    },
  ],
};

export const HA_NODES_MOCK_DEGRADED: GetHANodesResponse = {
  nodes: [
    {
      nodeName: 'pmm-ha-0',
      role: NodeRole.follower,
      status: 'alive',
    },
    {
      nodeName: 'pmm-ha-1',
      role: NodeRole.leader,
      status: 'alive',
    },
    {
      nodeName: 'pmm-ha-2',
      role: NodeRole.follower,
      status: 'dead',
    },
  ],
};

export const HA_NODES_MOCK_CRITICAL: GetHANodesResponse = {
  nodes: [
    {
      nodeName: 'pmm-ha-0',
      role: NodeRole.follower,
      status: 'dead',
    },
    {
      nodeName: 'pmm-ha-1',
      role: NodeRole.leader,
      status: 'alive',
    },
    {
      nodeName: 'pmm-ha-2',
      role: NodeRole.follower,
      status: 'dead',
    },
  ],
};

export const HA_NODES_MOCK_DOWN: GetHANodesResponse = {
  nodes: [
    {
      nodeName: 'pmm-ha-0',
      role: NodeRole.follower,
      status: 'suspect',
    },
    {
      nodeName: 'pmm-ha-1',
      role: NodeRole.leader,
      status: 'suspect',
    },
    {
      nodeName: 'pmm-ha-2',
      role: NodeRole.follower,
      status: 'suspect',
    },
  ],
};

const NODES_MOCK = [
  HA_NODES_MOCK_HEALTHY,
  HA_NODES_MOCK_DEGRADED,
  HA_NODES_MOCK_CRITICAL,
  HA_NODES_MOCK_DOWN,
];
let counter = 0;
export function getNodesMock() {
  const index = counter++;

  if (index >= NODES_MOCK.length - 1) {
    counter = 0;
  }

  return NODES_MOCK[index];
}
