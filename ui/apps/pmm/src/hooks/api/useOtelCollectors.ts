import {
  useMutation,
  useQuery,
  useQueryClient,
  UseMutationOptions,
  UseQueryOptions,
} from '@tanstack/react-query';
import {
  changeOtelCollectorLogSources,
  listInventoryNodes,
  listOtelCollectors,
  listPmmAgents,
  OtelCollectorAgent,
  OtelLogSource,
} from 'api/inventoryOtel';

export const OTEL_COLLECTORS_KEY = ['otelCollectors'] as const;
export const PMM_AGENTS_KEY = ['pmmAgents'] as const;
export const INVENTORY_NODES_KEY = ['inventoryNodes'] as const;

export const useOtelCollectors = (
  options?: Partial<UseQueryOptions<OtelCollectorAgent[]>>
) =>
  useQuery({
    queryKey: OTEL_COLLECTORS_KEY,
    queryFn: listOtelCollectors,
    ...options,
  });

export const usePmmAgents = (options?: Partial<UseQueryOptions<Awaited<ReturnType<typeof listPmmAgents>>>>) =>
  useQuery({
    queryKey: PMM_AGENTS_KEY,
    queryFn: listPmmAgents,
    ...options,
  });

export const useInventoryNodes = (
  options?: Partial<UseQueryOptions<Awaited<ReturnType<typeof listInventoryNodes>>>>
) =>
  useQuery({
    queryKey: INVENTORY_NODES_KEY,
    queryFn: listInventoryNodes,
    ...options,
  });

export const useChangeOtelCollectorLogSources = (
  options?: Partial<
    UseMutationOptions<OtelCollectorAgent, Error, { agentId: string; logSources: OtelLogSource[] }>
  >
) => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ agentId, logSources }) => changeOtelCollectorLogSources(agentId, logSources),
    onSuccess: (...args) => {
      qc.invalidateQueries({ queryKey: OTEL_COLLECTORS_KEY });
      options?.onSuccess?.(...args);
    },
    ...options,
  });
};
