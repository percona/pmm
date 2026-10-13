import type {
  ChangeRdsCredentialsParams,
  GetAgentVersionsResponse,
  ListAgentsParams,
  ManagedAgent,
  RdsExporter,
  RemoveAgentParams,
} from 'types/agent.types';
import { api } from './api';
import {
  type ListAgentsPayload,
  findRdsExporter,
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

export const getRdsExporter = async (
  nodeId: string
): Promise<RdsExporter | null> => {
  const res = await api.get<ListAgentsPayload>('/management/agents', {
    params: { nodeId },
    ...RAW_RESPONSE,
  });
  return findRdsExporter(res.data.agents);
};

export const changeRdsCredentials = async ({
  agentId,
  credentials,
}: ChangeRdsCredentialsParams) => {
  await api.put(`/inventory/agents/${agentId}`, { rdsExporter: credentials });
};
