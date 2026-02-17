import {
  useMutation,
  UseMutationOptions,
  useQuery,
  useQueryClient,
  UseQueryOptions,
} from '@tanstack/react-query';
import {
  getRunningSessions,
  searchQueries,
  startSession,
  stopSession,
} from 'api/rta';
import {
  RealtimeSession,
  StartSessionResponse,
  StartSessionPayload,
  StopSessionPayload,
  SearchQueriesPayload,
  QueryData,
} from 'types/rta.types';
import { ManagedService, ServiceType } from 'types/services.types';
import { useManagedServices } from './useServices';
import { useMemo } from 'react';
import { EmptyResponse } from 'types/util.types';

const KEYS = {
  LIST_SESSIONS: 'rta:list-sessions',
  START_SESSION: 'rta:start-session',
  START_SESSIONS: 'rta:start-sessions',
  STOP_SESSION: 'rta:stop-session',
  STOP_SESSIONS: 'rta:stop-sessions',
  SEARCH_QUERIES: 'rta:search-queries',
};

export const useRealtimeSessions = (
  options?: Partial<UseQueryOptions<RealtimeSession[]>>
) =>
  useQuery({
    queryKey: [KEYS.LIST_SESSIONS],
    queryFn: () => getRunningSessions(),
    ...options,
  });

export const useStartSession = (
  options?: Partial<
    UseMutationOptions<StartSessionResponse, Error, StartSessionPayload>
  >
) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationKey: [KEYS.START_SESSION],
    mutationFn: startSession,
    ...options,
    onSuccess: async (data, variables, onMutate, context) => {
      await options?.onSuccess?.(data, variables, onMutate, context);
      await queryClient.invalidateQueries({ queryKey: [KEYS.LIST_SESSIONS] });
    },
  });
};

export const useStartSessions = (
  options?: Partial<UseMutationOptions<StartSessionResponse[], Error, string[]>>
) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationKey: [KEYS.START_SESSIONS],
    mutationFn: (serviceIds: string[]) =>
      Promise.all(serviceIds.map((serviceId) => startSession({ serviceId }))),
    ...options,
    onSuccess: async (data, variables, onMutate, context) => {
      await options?.onSuccess?.(data, variables, onMutate, context);
      await queryClient.invalidateQueries({ queryKey: [KEYS.LIST_SESSIONS] });
    },
  });
};

export const useStopSession = (
  options?: Partial<
    UseMutationOptions<EmptyResponse, Error, StopSessionPayload>
  >
) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationKey: [KEYS.STOP_SESSION],
    mutationFn: stopSession,
    ...options,
    onSuccess: async (data, variables, onMutate, context) => {
      await options?.onSuccess?.(data, variables, onMutate, context);
      await queryClient.invalidateQueries({ queryKey: [KEYS.LIST_SESSIONS] });
    },
  });
};

export const useStopSessions = (
  options?: Partial<UseMutationOptions<EmptyResponse[], Error, string[]>>
) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationKey: [KEYS.STOP_SESSIONS],
    mutationFn: async (serviceIds: string[]) =>
      Promise.all(serviceIds.map((serviceId) => stopSession({ serviceId }))),
    ...options,
    onSuccess: async (data, variables, onMutate, context) => {
      await options?.onSuccess?.(data, variables, onMutate, context);
      await queryClient.invalidateQueries({ queryKey: [KEYS.LIST_SESSIONS] });
    },
  });
};

/**
 * Hook to get MongoDB services that don't have running RTA agents
 */
export const useAvailableServices = () => {
  const { data: sessions, isLoading: isLoadingSessions } =
    useRealtimeSessions();
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
    isLoading: isLoadingSessions || isLoadingServices,
    services: services?.services || [],
    sessions: sessions || [],
  };
};

export const useRealtimeQueries = (
  payload: SearchQueriesPayload,
  options?: Partial<UseQueryOptions<QueryData[]>>
) =>
  useQuery<QueryData[], Error, QueryData[]>({
    queryKey: [KEYS.SEARCH_QUERIES, payload],
    queryFn: async () => (await searchQueries(payload)).queries,
    ...options,
  });
