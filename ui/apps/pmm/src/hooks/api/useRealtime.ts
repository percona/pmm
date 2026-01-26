import { useMemo } from 'react';
import { useQuery, useMutation, useQueryClient, UseQueryOptions, UseMutationOptions } from '@tanstack/react-query';
import { listRunningRealtimeAgents, changeRealtimeAnalytics } from 'api/realtime';
import { useManagedServices } from 'hooks/api/useServices';
import {
  ListRunningRealtimeAgentsRequest,
  ListRunningRealtimeAgentsResponse,
  ChangeRealtimeAnalyticsRequest,
  ChangeRealtimeAnalyticsResponse,
} from 'types/realtime.types';
import { ManagedService, ServiceType } from 'types/services.types';

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

/**
 * Hook to get MongoDB services that don't have running RTA agents
 */
export const useAvailableServices = () => {
  const { data: runningAgentsData, isLoading: isLoadingAgents } = useRunningRealtimeAgents();
  const { data: servicesData, isLoading: isLoadingServices } = useManagedServices({
    serviceType: ServiceType.mongodb,
  });

  const availableServices = useMemo<ManagedService[]>(() => {
    if (!servicesData?.services) {
      return [];
    }

    const runningServiceIds = runningAgentsData?.agents?.map(
      (agent) => agent.serviceId
    ) ?? [];

    // Filter out services that already have running RTA agents
    return servicesData.services.filter(
      (service) => !runningServiceIds.includes(service.serviceId)
    );
  }, [servicesData, runningAgentsData]);

  return {
    availableServices,
    isLoading: isLoadingAgents || isLoadingServices,
    servicesData,
    runningAgentsData,
  };
};
