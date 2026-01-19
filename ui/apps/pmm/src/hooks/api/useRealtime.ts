import { useQuery, useMutation, useQueryClient, UseQueryOptions, UseMutationOptions } from '@tanstack/react-query';
import { listRunningRealtimeAgents, changeRealtimeAnalytics } from 'api/realtime';
import {
  ListRunningRealtimeAgentsRequest,
  ListRunningRealtimeAgentsResponse,
  ChangeRealtimeAnalyticsRequest,
  ChangeRealtimeAnalyticsResponse,
} from 'types/realtime.types';

export const REALTIME_AGENTS_QUERY_KEY = 'realtime:agents';

export const useRunningRealtimeAgents = (
  params?: ListRunningRealtimeAgentsRequest,
  options?: Partial<UseQueryOptions<ListRunningRealtimeAgentsResponse>>
) =>
  useQuery({
    queryKey: [REALTIME_AGENTS_QUERY_KEY, params],
    queryFn: () => listRunningRealtimeAgents(params),
    ...options,
  });

export const useChangeRealtimeAnalytics = (
  options?: Partial<UseMutationOptions<ChangeRealtimeAnalyticsResponse, Error, ChangeRealtimeAnalyticsRequest>>
) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: changeRealtimeAnalytics,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [REALTIME_AGENTS_QUERY_KEY] });
    },
    ...options,
  });
};
