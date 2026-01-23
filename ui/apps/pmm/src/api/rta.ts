import {
  ChangeRealTimeAgentPayload,
  ChangeRealTimeAgentResponse,
  ListRunningRealTimeAgentsResponse,
  ListRunningSessionsResponse,
  RealTimeSession,
  RunningRealTimeAgent,
  StartSessionPayload,
  StartSessionResponse,
  StopSessionPayload,
  StopSessionResponse,
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
  if (payload.enable) {
    const res = await api.post<ChangeRealTimeAgentResponse>(
      '/realtimeanalytics/sessions:start',
      payload
    );
    return res.data;
  }


  const res = await api.post<ChangeRealTimeAgentResponse>(
    '/realtimeanalytics/sessions:stop',
    payload
  );
  return res.data;
};

export const getRunningSessions = async (): Promise<RealTimeSession[]> => {
  const res = await api.get<ListRunningSessionsResponse>('/realtimeanalytics/sessions');
  return res.data.sessions;
};

export const startSession = async (payload: StartSessionPayload): Promise<StartSessionResponse> => {
  const res = await api.post<StartSessionResponse>(
    '/realtimeanalytics/sessions:start',
    payload
  );
  return res.data;
};

export const stopSession = async (payload: StopSessionPayload): Promise<StopSessionResponse> => {
  const res = await api.post<{}>(
    '/realtimeanalytics/sessions:stop',
    payload
  );
  return res.data;
};