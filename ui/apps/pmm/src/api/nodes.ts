import type { ListNodeTypesResponse } from 'types/nodes.types';
import { api } from './api';

export const getNodeTypes = async (): Promise<ListNodeTypesResponse> => {
  const res = await api.post('/inventory/nodes:getTypes');
  return res.data;
};
