import {
  type UseMutationOptions,
  type UseQueryOptions,
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query';
import { listNodes, removeNode } from 'api/nodes';
import type { ManagedNode, RemoveNodeParams } from 'types/nodes.types';
import { invalidateInventory, removeEach } from 'utils/inventory.utils';

export const useNodes = (options?: Partial<UseQueryOptions<ManagedNode[]>>) =>
  useQuery({
    queryKey: ['nodes:list'],
    queryFn: () => listNodes(),
    ...options,
  });

export const useRemoveNodes = (
  options?: Partial<UseMutationOptions<number, Error, RemoveNodeParams[]>>
) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (nodes: RemoveNodeParams[]) => removeEach(nodes, removeNode),
    ...options,
    onSuccess: async (data, variables, onMutate, context) => {
      await options?.onSuccess?.(data, variables, onMutate, context);
      await invalidateInventory(queryClient);
    },
  });
};
