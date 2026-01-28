/**
 * @fileoverview
 * This file contains the fallback API for the RTA API.
 * It is used to get the running real-time agents and change the real-time agent status.
 * It is used to fallback to the old API until the new API is merged (https://github.com/percona/pmm/pull/4956).
 */

import { AgentStatus } from 'types/agent.types';
import { api } from './api';

interface ListRunningRealtimeAgentsResponse {
  agents: RunningRealtimeAgent[];
}

interface RunningRealtimeAgent {
  agentId: string;
  serviceId: string;
  serviceName: string;
  cluster: string;
  startedAt: string;
  status: AgentStatus;
}

interface ChangeRealtimeAgentPayload {
  serviceId: string;
  enable: boolean;
}

interface ChangeRealtimeAgentResponse { }

/**
 * @deprecated use getRunningSessions instead
 */
export const getRunningRealtimeAgents = async (): Promise<
  RunningRealtimeAgent[]
> => {
  const res =
    await api.get<ListRunningRealtimeAgentsResponse>('/realtime/agents');
  return res.data.agents;
};

/**
 * @deprecated use startSession/stopSession instead
 */
export const changeRealtimeAgent = async (
  payload: ChangeRealtimeAgentPayload
): Promise<ChangeRealtimeAgentResponse> => {
  const res = await api.post<ChangeRealtimeAgentResponse>(
    '/realtime/change',
    payload
  );
  return res.data;
};
