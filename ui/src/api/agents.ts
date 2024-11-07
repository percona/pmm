import { GetAgentVersionsResponse } from 'types/agent.types';
import { api } from './api';

export const getAgentVersions = async () => {
  const res = await api.get<GetAgentVersionsResponse>(
    '/management/agents/versions'
  );
  return res.data.agentVersions;
};
