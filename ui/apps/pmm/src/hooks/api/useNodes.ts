import { useQuery, type UseQueryOptions } from '@tanstack/react-query';
import { getNodeTypes } from 'api/nodes';
import type { ListNodeTypesResponse } from 'types/nodes.types';

export const useNodeTypes = (
  options?: Partial<UseQueryOptions<ListNodeTypesResponse>>
) =>
  useQuery({
    queryKey: ['nodes:getTypes'],
    queryFn: () => getNodeTypes(),
    ...options,
  });
