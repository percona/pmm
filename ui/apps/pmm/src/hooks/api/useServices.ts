import { useQuery, UseQueryOptions } from '@tanstack/react-query';
import { getServiceTypes, listServices } from 'api/services';
import {
  ListServicesParams,
  ListServicesResponse,
  ListTypesResponse,
} from 'types/services.types';

export const useServiceTypes = (
  options?: Partial<UseQueryOptions<ListTypesResponse>>
) =>
  useQuery({
    queryKey: ['services:getTypes'],
    queryFn: () => getServiceTypes(),
    ...options,
  });

export const useServices = (
  params: ListServicesParams = {},
  options?: Partial<UseQueryOptions<ListServicesResponse>>
) =>
  useQuery({
    queryKey: ['services:list', params],
    queryFn: ({ queryKey: [, params] }) =>
      listServices(params as ListServicesParams),
    ...options,
  });
