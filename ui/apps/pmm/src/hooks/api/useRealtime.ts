import {
  useMutation,
  UseMutationOptions,
  useQuery,
  useQueryClient,
  UseQueryOptions,
} from '@tanstack/react-query';
import { getRunningSessions, startSession, stopSession } from 'api/rta';
import {
  RealTimeSession,
  StartSessionResponse,
  StartSessionPayload,
  StopSessionPayload,
  StopSessionResponse,
} from 'types/rta.types';
import { useManagedServices } from './useServices';
import { ManagedService, ServiceType } from 'types/services.types';
import { useMemo } from 'react';

const KEYS = {
  LIST_SESSIONS: 'rta:list-sessions',
  START_SESSION: 'rta:start-session',
  START_SESSIONS: 'rta:start-sessions',
  STOP_SESSION: 'rta:stop-session',
  STOP_SESSIONS: 'rta:stop-sessions',

}

export const useRealTimeSessions = (
  options?: Partial<UseQueryOptions<RealTimeSession[]>>
) =>
  useQuery({
    queryKey: [KEYS.LIST_SESSIONS],
    queryFn: () => getRunningSessions(),
    ...options,
  });

export const useStartSession = (
  options?: Partial<UseMutationOptions<StartSessionResponse, Error, StartSessionPayload>>
) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationKey: [KEYS.START_SESSION],
    mutationFn: startSession,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [KEYS.LIST_SESSIONS] }),

    ...options,
  });
}

export const useStartSessions = (
  options?: Partial<UseMutationOptions<StartSessionResponse[], Error, string[]>>
) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationKey: [KEYS.START_SESSIONS],
    mutationFn: (serviceIds: string[]) => Promise.all(
      serviceIds.map((serviceId) => startSession({ serviceId }))
    ),
    ...options,
    onSuccess: (data, variables, context) => {
      options?.onSuccess?.(data, variables, context);
      return queryClient.invalidateQueries({ queryKey: [KEYS.LIST_SESSIONS] })
    },
  });
}

export const useStopSession = (
  options?: Partial<UseMutationOptions<StopSessionResponse, Error, StopSessionPayload>>
) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationKey: [KEYS.STOP_SESSION],
    mutationFn: stopSession,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [KEYS.LIST_SESSIONS] }),
    ...options,
  })
    ;
}

export const useStopSessions = (
  options?: Partial<UseMutationOptions<StopSessionResponse[], Error, string[]>>
) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationKey: [KEYS.STOP_SESSIONS],
    mutationFn: async (serviceIds: string[]) => Promise.all(
      serviceIds.map((serviceId) => stopSession({ serviceId }))
    ),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: [KEYS.LIST_SESSIONS] }),
    ...options,
  });
}

/**
 * Hook to get MongoDB services that don't have running RTA agents
 */
export const useAvailableServices = () => {
  const { data: sessionsData, isLoading: isLoadingSessions } = useRealTimeSessions();
  const { data: servicesData, isLoading: isLoadingServices } = useManagedServices({
    serviceType: ServiceType.mongodb,
  });

  const availableServices = useMemo<ManagedService[]>(() => {
    if (!servicesData?.services) {
      return [];
    }

    const runningServiceIds = sessionsData?.map(
      (session) => session.serviceId
    ) ?? [];

    // Filter out services that already have running RTA agents
    return servicesData.services.filter(
      (service) => !runningServiceIds.includes(service.serviceId)
    );
  }, [servicesData, sessionsData]);

  return {
    availableServices,
    isLoading: isLoadingSessions || isLoadingServices,
    servicesData,
    sessionsData,
  };
};
