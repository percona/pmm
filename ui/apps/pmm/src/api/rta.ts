import {
  ListRunningSessionsResponse,
  RealtimeSession,
  RealtimeSessionStatus,
  SearchQueriesPayload,
  SearchQueriesResponse,
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
    // const agents = await getRunningRealtimeAgents();
    // return agents.map<RealtimeSession>((agent) => ({
    //   status: RealtimeSessionStatus.unspecified,
    //   startTime: agent.startedAt,
    //   serviceId: agent.serviceId,
    //   serviceName: agent.serviceName,
    //   clusterName: agent.cluster,
    // }));
    return [
      {
        serviceId: '1',
        serviceName: 'Service 1',
        clusterName: 'Cluster 1',
        startTime: '2021-01-01T00:00:00Z',
        status: RealtimeSessionStatus.unspecified,
      },
    ];
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

export const searchQueries = async (
  payload: SearchQueriesPayload
): Promise<SearchQueriesResponse> => {
  // const res = await api.post<SearchQueriesResponse>(
  //   '/realtimeanalytics/queries:search',
  //   payload
  // );
  // return res.data;

  return {
    queries: [
      {
        serviceId: '1',
        serviceName: 'Service 1',
        queryId: '1',
        queryText: '{ find: "mycollection", filter: { status: "active" } }',
        queryExecutionDuration: '1000',
        queryRawJson: '{"query": "SELECT * FROM users"}',
        queryCollectTime: '2021-01-01T00:00:00Z',
        clientAddress: '127.0.0.1',
        mongoDbPayload: {
          operation: 'find',
          collection: 'users',
          databaseName: 'test',
          clientAppName: 'test',
          dbInstanceAddress: '127.0.0.1',
          operationStartTime: '2021-01-01T00:00:00Z',
          username: 'test',
          planSummary: 'COLLSCAN',
        },
      },
    ],
  };
};
