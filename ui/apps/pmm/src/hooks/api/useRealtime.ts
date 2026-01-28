import {
  useMutation,
  UseMutationOptions,
  useQuery,
  useQueryClient,
  UseQueryOptions,
} from '@tanstack/react-query';
import { getRunningSessions, startSession, stopSession } from 'api/rta';
import {
  RealtimeSession,
  StartSessionResponse,
  StartSessionPayload,
  StopSessionPayload,
  StopSessionResponse,
} from 'types/rta.types';
import { ManagedService, ServiceType } from 'types/services.types';
import { useManagedServices } from './useServices';
import { useMemo } from 'react';

const KEYS = {
  LIST_SESSIONS: 'rta:list-sessions',
  START_SESSION: 'rta:start-session',
  START_SESSIONS: 'rta:start-sessions',
  STOP_SESSION: 'rta:stop-session',
  STOP_SESSIONS: 'rta:stop-sessions',
}

export const useRealtimeSessions = (
  options?: Partial<UseQueryOptions<RealtimeSession[]>>
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
    onSuccess: async (data, variables, context) => {
      await options?.onSuccess?.(data, variables, context);
      await queryClient.invalidateQueries({ queryKey: [KEYS.LIST_SESSIONS] })
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
  const { data: sessions, isLoading: isLoadingSesssions } = useRealtimeSessions();
  const { data: services, isLoading: isLoadingServices } = useManagedServices({
    serviceType: ServiceType.mongodb,
  });

  const availableServices = useMemo<ManagedService[]>(() => {
    if (!services?.services) {
      return [];
    }

    const runningServiceIds = (sessions || []).map(
      (session) => session.serviceId
    );

    // Filter out services that already have running RTA agents
    return services.services.filter(
      (service) => !runningServiceIds.includes(service.serviceId)
    );
  }, [services, sessions]);

  return {
    availableServices,
    isLoading: isLoadingSesssions || isLoadingServices,
    services: services?.services || [],
    sessions: sessions || [],
  };
};