import {
  AvailableServicesResponse,
  ListRunningSessionsResponse,
  RealtimeSession,
  SearchQueriesPayload,
  SearchQueriesResponse,
  StartSessionPayload,
  StartSessionResponse,
  StopSessionPayload,
} from 'types/rta.types';
import { api } from './api';
import { EmptyResponse } from 'types/util.types';

export const getRunningSessions = async (): Promise<RealtimeSession[]> => {
  const res = await api.get<ListRunningSessionsResponse>(
    '/realtimeanalytics/sessions'
  );
  return res.data.sessions;
};

export const startSession = async (
  payload: StartSessionPayload
): Promise<StartSessionResponse> => {
  const res = await api.post<StartSessionResponse>(
    '/realtimeanalytics/sessions:start',
    payload
  );
  return res.data;
};

export const stopSession = async (
  payload: StopSessionPayload
): Promise<EmptyResponse> => {
  const res = await api.post<EmptyResponse>(
    '/realtimeanalytics/sessions:stop',
    payload
  );
  return res.data;
};

export const searchQueries = async (
  payload: SearchQueriesPayload
): Promise<SearchQueriesResponse> => {
  const res = await api.post<SearchQueriesResponse>(
    '/realtimeanalytics/queries:search',
    payload
  );
  return res.data;
};

export const getAvailableServices = async (): Promise<
  AvailableServicesResponse['mongodb']
> => {
  const res = await api.get<AvailableServicesResponse>(
    '/realtimeanalytics/services'
  );
  return res.data.mongodb;
};
