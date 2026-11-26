import { GetHANodesResponse, GetHAStatusResponse } from 'types/ha.types';
import { api } from './api';

export const getHAStatus = async (): Promise<GetHAStatusResponse> => {
  const response = await api.get<GetHAStatusResponse>('/ha/status');
  return response.data;
};

export const getHANodes = async (): Promise<GetHANodesResponse> => {
  const response = await api.get<GetHANodesResponse>('/ha/nodes');
  return response.data;
};
