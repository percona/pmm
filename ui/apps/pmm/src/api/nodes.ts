import type {
  ManagedNode,
  RemoveNodeParams,
  RemoveNodeResponse,
} from 'types/nodes.types';
import { api } from './api';
import {
  type ListNodesPayload,
  RAW_RESPONSE,
  toManagedNode,
} from './inventory.utils';

export const listNodes = async (): Promise<ManagedNode[]> => {
  const res = await api.get<ListNodesPayload>(
    '/management/nodes',
    RAW_RESPONSE
  );
  return (res.data.nodes ?? []).map(toManagedNode);
};

export const removeNode = async ({ nodeId, force }: RemoveNodeParams) => {
  const res = await api.delete<RemoveNodeResponse>(
    `/inventory/nodes/${nodeId}`,
    { params: { force } }
  );
  return res.data.warning;
};
