import {
  ListRunningSessionsResponse,
  RealtimeSession,
  RealtimeSessionStatus,
  StartSessionPayload,
  StartSessionResponse,
  StopSessionPayload,
} from 'types/rta.types';
import { api } from './api';
import { changeRealtimeAgent, getRunningRealtimeAgents } from './rta.fallback';
import { EmptyResponse } from 'types/util.types';

export const getRunningSessions = async (): Promise<RealtimeSession[]> => {
  try {
    const res = await api.get<ListRunningSessionsResponse>(
      '/realtimeanalytics/sessions'
    );
    return res.data.sessions;
  } catch (error) {
    // todo: temporary fallback till https://github.com/percona/pmm/pull/4956 gets merged
    const agents = await getRunningRealtimeAgents();
    return agents.map<RealtimeSession>((agent) => ({
      status: RealtimeSessionStatus.unspecified,
      startTime: agent.startedAt,
      serviceId: agent.serviceId,
      serviceName: agent.serviceName,
      clusterName: agent.cluster,
    }));
  }
};

export const startSession = async (
  payload: StartSessionPayload
): Promise<StartSessionResponse> => {
  try {
    const res = await api.post<StartSessionResponse>(
      '/realtimeanalytics/sessions:start',
      payload
    );
    return res.data;
  } catch (error) {
    // todo: temporary fallback till https://github.com/percona/pmm/pull/4956 gets merged
    await changeRealtimeAgent({
      serviceId: payload.serviceId,
      enable: true,
    });
    return {
      session: {
        serviceId: payload.serviceId,
        serviceName: '',
        clusterName: '',
        startTime: new Date().toISOString(),
        status: RealtimeSessionStatus.unspecified,
      },
    };
  }
};

export const stopSession = async (
  payload: StopSessionPayload
): Promise<EmptyResponse> => {
  try {
    const res = await api.post<EmptyResponse>(
      '/realtimeanalytics/sessions:stop',
      payload
    );
    return res.data;
  } catch (error) {
    // todo: temporary fallback till https://github.com/percona/pmm/pull/4956 gets merged
    await changeRealtimeAgent({
      serviceId: payload.serviceId,
      enable: false,
    });
    return {};
  }
};
