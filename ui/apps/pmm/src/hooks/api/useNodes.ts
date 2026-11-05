import { useQuery, type UseQueryOptions } from '@tanstack/react-query';
import { listNodes } from 'api/nodes';
import type { ListNodesParams, ListNodesResponse } from 'types/nodes.types';

export const useNodes = (
  params: ListNodesParams = {},
  options?: Partial<UseQueryOptions<ListNodesResponse>>
) =>
  useQuery({
    queryKey: ['nodes:list', params],
    queryFn: ({ queryKey: [, params] }) => listNodes(params as ListNodesParams),
    ...options,
  });
