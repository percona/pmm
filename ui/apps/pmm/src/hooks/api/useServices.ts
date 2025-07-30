import { useQuery, UseQueryOptions } from '@tanstack/react-query';
import { getServiceTypes } from 'api/services';
import { ListTypesResponse } from 'types/services.types';

export const useServiceTypes = (
  options?: Partial<UseQueryOptions<ListTypesResponse>>
) =>
  useQuery({
    queryKey: ['services:getTypes'],
    queryFn: () => getServiceTypes(),
    ...options,
  });
