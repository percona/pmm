import { useQuery, type UseQueryOptions } from '@tanstack/react-query';
import { listNodes } from 'api/nodes';
import type { ListNodesResponse } from 'types/nodes.types';

export const NODES_QUERY_KEY = ['nodes:list'] as const;

export const useNodes = (
  options?: Partial<UseQueryOptions<ListNodesResponse>>
) =>
  useQuery({
    queryKey: NODES_QUERY_KEY,
    queryFn: () => listNodes(),
    ...options,
  });
