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
  expectedNodes: 3,
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
  expectedNodes: 3,
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
  expectedNodes: 3,
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
  expectedNodes: 3,
};

export const getHAStatus = async (): Promise<GetHAStatusResponse> =>
  Promise.resolve(HA_STATUS_MOCK);

export const getHANodes = async (): Promise<GetHANodesResponse> =>
  Promise.resolve(HA_NODES_MOCK_HEALTHY);
