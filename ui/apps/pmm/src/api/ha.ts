import { GetHANodesResponse, GetHAStatusResponse } from 'types/ha.types';
import { api } from './api';
import { getNodesMock, HA_STATUS_MOCK } from './ha.mock';

export const getHAStatus = async (): Promise<GetHAStatusResponse> => {
  try {
    const response = await api.get<GetHAStatusResponse>('/ha/status');
    return response.data;
    // todo: remove mock data
  } catch {
    return HA_STATUS_MOCK;
  }
};

export const getHANodes = async (): Promise<GetHANodesResponse> => {
  try {
    const response = await api.get<GetHANodesResponse>('/ha/nodes');
    return response.data;
    // todo: remove mock data
  } catch {
    return getNodesMock();
  }
};
