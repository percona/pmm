import {
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
