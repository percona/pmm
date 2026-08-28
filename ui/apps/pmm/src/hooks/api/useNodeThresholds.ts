import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type {
  UseMutationOptions,
  UseQueryOptions,
} from "@tanstack/react-query";
import { batchUpdateThresholds, getThresholds } from "api/alerting";
import type {
  BatchUpdateThresholdsResponse,
  ListThresholdsResponse,
  ThresholdUpdate,
} from "types/alerting.types";

export const nodeThresholdsQueryKey = (nodeId: string) => [
  "alerting:nodeThresholds",
  nodeId,
];

// Asking for a target returns every overridable parameter for it, overridden or not,
// which is what the modal lists. Asking without one would return only existing
// overrides.
export const useNodeThresholds = (
  nodeId: string,
  options?: Partial<UseQueryOptions<ListThresholdsResponse>>,
) =>
  useQuery({
    queryKey: nodeThresholdsQueryKey(nodeId),
    queryFn: () => getThresholds("THRESHOLD_SCOPE_NODE", nodeId),
    enabled: !!nodeId,
    ...options,
  });

// One transactional call for a whole form's worth of edits. Firing a request per row
// would leave the modal half-applied on a partial failure, with no way to report
// which rows took effect.
export const useBatchUpdateNodeThresholds = (
  nodeId: string,
  options?: Partial<
    UseMutationOptions<BatchUpdateThresholdsResponse, Error, ThresholdUpdate[]>
  >,
) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationKey: ["alerting:batchUpdateNodeThresholds", nodeId],
    mutationFn: (updates: ThresholdUpdate[]) => batchUpdateThresholds(updates),
    ...options,
    onSuccess: async (data, variables, onMutate, context) => {
      await options?.onSuccess?.(data, variables, onMutate, context);
      await queryClient.invalidateQueries({
        queryKey: nodeThresholdsQueryKey(nodeId),
      });
    },
  });
};
