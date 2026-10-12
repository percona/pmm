import type {
  GetAgentVersionsResponse,
  ListAgentsParams,
  ManagedAgent,
  RemoveAgentParams,
} from 'types/agent.types';
import { api } from './api';
import {
  type ListAgentsPayload,
  RAW_RESPONSE,
  toManagedAgent,
} from './inventory.utils';

export const getAgentVersions = async () => {
  const res = await api.get<GetAgentVersionsResponse>(
    '/management/agents/versions'
  );
  return res.data.agentVersions;
};

export const listAgents = async (
  params: ListAgentsParams
): Promise<ManagedAgent[]> => {
  const res = await api.get<ListAgentsPayload>('/management/agents', {
    params,
    ...RAW_RESPONSE,
  });
  return (res.data.agents ?? []).map(toManagedAgent);
};

export const removeAgent = async ({ agentId, force }: RemoveAgentParams) => {
  await api.delete(`/inventory/agents/${agentId}`, { params: { force } });
};
