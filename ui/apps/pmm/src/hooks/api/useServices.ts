import { useQuery, UseQueryOptions } from '@tanstack/react-query';
import { getServiceTypes, listServices, listManagedServices } from 'api/services';
import {
  ListServicesParams,
  ListServicesResponse,
  ListTypesResponse,
  ManagedServicesResponse,
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

export const useManagedServices = (
  params: ListServicesParams = {},
  options?: Partial<UseQueryOptions<ManagedServicesResponse>>
) =>
  useQuery({
    queryKey: ['services:managed', params],
    queryFn: ({ queryKey: [, params] }) =>
      listManagedServices(params as ListServicesParams),
    ...options,
  });
