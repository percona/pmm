import {
  ListRunningSessionsResponse,
  RealTimeSession,
  RealTimeSessionStatus,
  StartSessionPayload,
  StartSessionResponse,
  StopSessionPayload,
  StopSessionResponse,
} from 'types/rta.types';
import { api } from './api';
import { changeRealTimeAgent, getRunningRealTimeAgents } from './rta.fallback';


export const getRunningSessions = async (): Promise<RealTimeSession[]> => {
  try {
    const res = await api.get<ListRunningSessionsResponse>('/realtimeanalytics/sessions');
    return res.data.sessions;
  } catch (error) {
    // todo: temporary fallback till https://github.com/percona/pmm/pull/4956 gets merged
    const agents = await getRunningRealTimeAgents();
    return agents.map<RealTimeSession>((agent) => ({
      status: RealTimeSessionStatus.unspecified,
      startTime: agent.startedAt,
      serviceId: agent.serviceId,
      serviceName: agent.serviceName,
      clusterName: agent.cluster,
    }));
  }
};

export const startSession = async (payload: StartSessionPayload): Promise<StartSessionResponse> => {
  try {
    const res = await api.post<StartSessionResponse>(
      '/realtimeanalytics/sessions:start',
      payload
    );
    return res.data;
  } catch (error) {
    // todo: temporary fallback till https://github.com/percona/pmm/pull/4956 gets merged
    await changeRealTimeAgent({
      serviceId: payload.serviceId,
      enable: true,
    })
    return {
      session: {
        serviceId: payload.serviceId,
        serviceName: '',
        clusterName: '',
        startTime: new Date().toISOString(),
        status: RealTimeSessionStatus.unspecified,
      },
    };
  }
};

export const stopSession = async (payload: StopSessionPayload): Promise<StopSessionResponse> => {
  try {
    const res = await api.post<{}>(
      '/realtimeanalytics/sessions:stop',
      payload
    );
    return res.data;
  } catch (error) {
    // todo: temporary fallback till https://github.com/percona/pmm/pull/4956 gets merged
    await changeRealTimeAgent({
      serviceId: payload.serviceId,
      enable: false,
    })
    return {
      session: {
        serviceId: payload.serviceId,
        serviceName: '',
        clusterName: '',
        startTime: new Date().toISOString(),
        status: RealTimeSessionStatus.unspecified,
      },
    };
  }
};