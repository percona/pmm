import type { ListNodesResponse } from 'types/nodes.types';
import { api } from './api';

export const listNodes = async (): Promise<ListNodesResponse> => {
  const res = await api.get<ListNodesResponse>('/management/nodes');
  return res.data;
};
