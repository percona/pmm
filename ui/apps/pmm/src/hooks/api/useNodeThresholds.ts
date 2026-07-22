import {
  useMutation,
  UseMutationOptions,
  useQuery,
  useQueryClient,
  UseQueryOptions,
} from '@tanstack/react-query';
import {
  deleteNodeThreshold,
  getNodeThresholds,
  setNodeThreshold,
} from 'api/alerting';
import {
  ListNodeThresholdsResponse,
  SetNodeThresholdPayload,
} from 'types/alerting.types';

export const nodeThresholdsQueryKey = (nodeId: string) => [
  'alerting:nodeThresholds',
  nodeId,
];

export const useNodeThresholds = (
  nodeId: string,
  options?: Partial<UseQueryOptions<ListNodeThresholdsResponse>>
) =>
  useQuery({
    queryKey: nodeThresholdsQueryKey(nodeId),
    queryFn: () => getNodeThresholds(nodeId),
    enabled: !!nodeId,
    ...options,
  });

export const useSetNodeThreshold = (
  nodeId: string,
  options?: Partial<UseMutationOptions<unknown, Error, SetNodeThresholdPayload>>
) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationKey: ['alerting:setNodeThreshold', nodeId],
    mutationFn: (payload: SetNodeThresholdPayload) =>
      setNodeThreshold(nodeId, payload),
    ...options,
    onSuccess: async (data, variables, onMutate, context) => {
      await options?.onSuccess?.(data, variables, onMutate, context);
      await queryClient.invalidateQueries({
        queryKey: nodeThresholdsQueryKey(nodeId),
      });
    },
  });
};

export interface DeleteNodeThresholdVariables {
  ruleId: string;
  paramName: string;
}

export const useDeleteNodeThreshold = (
  nodeId: string,
  options?: Partial<
    UseMutationOptions<unknown, Error, DeleteNodeThresholdVariables>
  >
) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationKey: ['alerting:deleteNodeThreshold', nodeId],
    mutationFn: ({ ruleId, paramName }: DeleteNodeThresholdVariables) =>
      deleteNodeThreshold(nodeId, ruleId, paramName),
    ...options,
    onSuccess: async (data, variables, onMutate, context) => {
      await options?.onSuccess?.(data, variables, onMutate, context);
      await queryClient.invalidateQueries({
        queryKey: nodeThresholdsQueryKey(nodeId),
      });
    },
  });
};
