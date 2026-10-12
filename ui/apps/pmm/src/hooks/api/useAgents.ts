import {
  type UseMutationOptions,
  type UseQueryOptions,
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query';
import { getAgentVersions, listAgents, removeAgent } from 'api/agents';
import type {
  GetAgentVersionItem,
  ListAgentsParams,
  ManagedAgent,
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
