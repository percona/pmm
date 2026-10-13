import {
  type UseMutationOptions,
  type UseQueryOptions,
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query';
import {
  changeRdsCredentials,
  getAgentVersions,
  getRdsExporter,
  listAgents,
  removeAgent,
} from 'api/agents';
import type {
  ChangeRdsCredentialsParams,
  GetAgentVersionItem,
  ListAgentsParams,
  ManagedAgent,
  RdsExporter,
  RemoveAgentParams,
} from 'types/agent.types';
import { invalidateInventory, removeEach } from 'utils/inventory.utils';

export const useAgentVersions = (
  options?: Partial<UseQueryOptions<GetAgentVersionItem[]>>
) =>
  useQuery({
    queryKey: ['agent/versions'],
    queryFn: getAgentVersions,
    ...options,
  });

export const useAgents = (
  params: ListAgentsParams,
  options?: Partial<UseQueryOptions<ManagedAgent[]>>
) =>
  useQuery({
    queryKey: ['agents:list', params],
    queryFn: ({ queryKey: [, params] }) =>
      listAgents(params as ListAgentsParams),
    ...options,
  });

export const useRemoveAgents = (
  options?: Partial<UseMutationOptions<number, Error, RemoveAgentParams[]>>
) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (agents: RemoveAgentParams[]) =>
      removeEach(agents, removeAgent),
    ...options,
    onSuccess: async (data, variables, onMutate, context) => {
      await options?.onSuccess?.(data, variables, onMutate, context);
      await invalidateInventory(queryClient);
    },
  });
};

export const useRdsExporter = (
  nodeId?: string,
  options?: Partial<UseQueryOptions<RdsExporter | null>>
) =>
  useQuery({
    queryKey: ['agents:rdsExporter', nodeId],
    queryFn: () => getRdsExporter(nodeId as string),
    enabled: !!nodeId,
    ...options,
  });

export const useChangeRdsCredentials = (
  options?: Partial<UseMutationOptions<void, Error, ChangeRdsCredentialsParams>>
) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (params: ChangeRdsCredentialsParams) =>
      changeRdsCredentials(params),
    ...options,
    onSuccess: async (data, variables, onMutate, context) => {
      await options?.onSuccess?.(data, variables, onMutate, context);
      await invalidateInventory(queryClient);
    },
  });
};
