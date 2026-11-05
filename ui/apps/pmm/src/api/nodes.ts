import type { ListNodesParams, ListNodesResponse } from 'types/nodes.types';
import { api } from './api';

export const listNodes = async (
  params: ListNodesParams
): Promise<ListNodesResponse> => {
  const res = await api.get('/inventory/nodes', { params });
  return res.data;
};
