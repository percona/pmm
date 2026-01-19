import {
  ChangeRealtimeAnalyticsRequest,
  ChangeRealtimeAnalyticsResponse,
  ListRunningRealtimeAgentsRequest,
  ListRunningRealtimeAgentsResponse,
} from 'types/realtime.types';
import { api } from './api';

export const listRunningRealtimeAgents = async (
  params?: ListRunningRealtimeAgentsRequest
): Promise<ListRunningRealtimeAgentsResponse> => {
  const res = await api.get('/realtime/agents', { params });
  return res.data;
};

export const changeRealtimeAnalytics = async (
  data: ChangeRealtimeAnalyticsRequest
): Promise<ChangeRealtimeAnalyticsResponse> => {
  const res = await api.post('/realtime/change', data);
  return res.data;
};
