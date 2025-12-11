import {
  GetHANodesResponse,
  GetHAStatusResponse,
  NodeRole,
} from 'types/ha.types';
import { api } from './api';

export const getHAStatus = async (): Promise<GetHAStatusResponse> => {
  try {
    const response = await api.get<GetHAStatusResponse>('/ha/status');
    return response.data;
    // todo: remove mock data
  } catch {
    return {
      status: 'Enabled',
    };
  }
};

export const getHANodes = async (): Promise<GetHANodesResponse> => {
  try {
    const response = await api.get<GetHANodesResponse>('/ha/nodes');
    return response.data;
    // todo: remove mock data
  } catch {
    return {
      nodes: [
        {
          nodeName: 'pmm-ha-1',
          role: NodeRole.follower,
          status: 'alive',
        },
        {
          nodeName: 'pmm-ha-0',
          role: NodeRole.leader,
          status: 'alive',
        },
        {
          nodeName: 'pmm-ha-2',
          role: NodeRole.follower,
          status: 'alive',
        },
        {
          nodeName: 'pmm-ha-3',
          role: NodeRole.follower,
          status: 'alive',
        },
        {
          nodeName: 'pmm-ha-4',
          role: NodeRole.follower,
          status: 'alive',
        },
        {
          nodeName: 'pmm-ha-5',
          role: NodeRole.follower,
          status: 'alive',
        },
      ],
    };
  }
};
