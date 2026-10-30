import {
  type UseMutationOptions,
  type UseQueryOptions,
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query';
import { getNode, listNodes, removeNode } from 'api/nodes';
import type {
  ManagedNode,
  RemoveNodeParams,
  RemoveNodesResult,
} from 'types/nodes.types';
import { invalidateInventory, removeEach } from 'utils/inventory.utils';

export const useNodes = (options?: Partial<UseQueryOptions<ManagedNode[]>>) =>
  useQuery({
    queryKey: ['nodes:list'],
    queryFn: () => listNodes(),
    ...options,
  });

export const useNode = (
  nodeId?: string,
  options?: Partial<UseQueryOptions<ManagedNode | null>>
) =>
  useQuery({
    queryKey: ['nodes:get', nodeId],
    queryFn: () => getNode(nodeId as string),
    enabled: !!nodeId,
    ...options,
  });

export const useRemoveNodes = (
  options?: Partial<
    UseMutationOptions<RemoveNodesResult, Error, RemoveNodeParams[]>
  >
) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (nodes: RemoveNodeParams[]) => {
      const warnings: string[] = [];
      const deleted = await removeEach(nodes, async (node) => {
        const warning = await removeNode(node);

        if (warning) {
          warnings.push(warning);
        }
      });

      return { deleted, warnings };
    },
    ...options,
    onSuccess: async (data, variables, onMutate, context) => {
      await options?.onSuccess?.(data, variables, onMutate, context);
      await invalidateInventory(queryClient);
    },
  });
};
