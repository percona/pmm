import {
  ChangeRealTimeAgentPayload,
  ChangeRealTimeAgentResponse,
  ListRunningRealTimeAgentsResponse,
  RunningRealTimeAgent,
} from 'types/rta.types';
import { api } from './api';

export const getRunningRealTimeAgents = async (): Promise<
  RunningRealTimeAgent[]
> => {
  const res =
    await api.get<ListRunningRealTimeAgentsResponse>('/realtime/agents');
  return res.data.agents;
};

export const changeRealTimeAgent = async (
  payload: ChangeRealTimeAgentPayload
): Promise<ChangeRealTimeAgentResponse> => {
  const res = await api.post<ChangeRealTimeAgentResponse>(
    '/realtime/change',
    payload
  );
  return res.data;
};
