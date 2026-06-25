import {
  useMutation,
  UseMutationOptions,
  useQuery,
  useQueryClient,
  UseQueryOptions,
} from '@tanstack/react-query';
import {
  getAvailableServices,
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
  RawQueryData,
} from 'types/rta.types';
import { ServiceType, VersionedService } from 'types/services.types';
import { useMemo } from 'react';
import { EmptyResponse } from 'types/util.types';
import { parseDuration } from 'utils/duration.utils';
import { useUser } from 'contexts/user';

const mapQueryData = (query: RawQueryData): QueryData => {
  const transactionDurationMs = query.postgresqlPayload?.transactionStartTime
    ? (Date.now() - new Date(query.postgresqlPayload.transactionStartTime).getTime()) / 1000
    : null;

  return {
    ...query,
    queryExecutionDurationMs: query.queryExecutionDuration
      ? parseDuration(query.queryExecutionDuration) / 1000
      : null,
    transactionDurationMs,
  };
};

const collapseParallelWorkers = (queries: QueryData[]): QueryData[] => {
  const leaders = new Map<number, string>();

  queries.forEach((query) => {
    const pid = query.postgresqlPayload?.pid;
    if (pid && query.postgresqlPayload?.backendType !== 'parallel worker') {
      leaders.set(pid, query.queryId);
    }
  });

  return queries.filter((query) => {
    const payload = query.postgresqlPayload;
    if (!payload || payload.backendType !== 'parallel worker' || !payload.leaderPid) {
      return true;
    }

    return false;
  }).map((query) => {
    const payload = query.postgresqlPayload;
    if (payload?.backendType === 'parallel worker' && payload.leaderPid) {
      return {
        ...query,
        isParallelWorker: true,
        leaderQueryId: leaders.get(payload.leaderPid),
      };
    }

    return query;
  });
};

const KEYS = {
  LIST_SESSIONS: 'rta:list-sessions',
  START_SESSION: 'rta:start-session',
  START_SESSIONS: 'rta:start-sessions',
  STOP_SESSION: 'rta:stop-session',
  STOP_SESSIONS: 'rta:stop-sessions',
  SEARCH_QUERIES: 'rta:search-queries',
  AVAILABLE_SERVICES: 'rta:available-services',
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
export const useAvailableServices = (serviceTypes?: ServiceType[]) => {
  const { user } = useUser();
  const { data: sessions, isLoading: isLoadingSessions } =
    useRealtimeSessions();
  const { data: services = { mongodb: [] }, isLoading: isLoadingServices } =
    useQuery({
      queryKey: [KEYS.AVAILABLE_SERVICES],
      queryFn: () => getAvailableServices(serviceTypes),
      enabled: !!user,
    });

  const availableServices = useMemo<VersionedService[]>(() => {
    const runningServiceIds = (sessions || []).map(
      (session) => session.serviceId
    );

    // Filter out services that already have running RTA agents
    return Object.values(services)
      .flat()
      .filter((service) => !runningServiceIds.includes(service.serviceId));
  }, [services, sessions]);

  return {
    availableServices,
    isLoading: isLoadingSessions || isLoadingServices,
    services,
    sessions: sessions || [],
  };
};

export const useRealtimeQueries = (
  payload: SearchQueriesPayload,
  options?: Partial<UseQueryOptions<RawQueryData[]>>
) =>
  useQuery<RawQueryData[], Error, QueryData[]>({
    queryKey: [KEYS.SEARCH_QUERIES, payload],
    queryFn: async () => (await searchQueries(payload)).queries,
    select: (data) => collapseParallelWorkers(data.map(mapQueryData)),
    ...options,
  });
