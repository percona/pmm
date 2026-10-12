import {
  type UseMutationOptions,
  type UseQueryOptions,
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query';
import {
  getServiceTypes,
  listServices,
  listManagedServices,
  removeService,
} from 'api/services';
import type {
  ListServicesParams,
  ListServicesResponse,
  ListTypesResponse,
  ManagedServicesResponse,
  RemoveServiceParams,
} from 'types/services.types';
import { invalidateInventory, removeEach } from 'utils/inventory.utils';

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

export const useRemoveServices = (
  options?: Partial<UseMutationOptions<number, Error, RemoveServiceParams[]>>
) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (services: RemoveServiceParams[]) =>
      removeEach(services, removeService),
    ...options,
    onSuccess: async (data, variables, onMutate, context) => {
      await options?.onSuccess?.(data, variables, onMutate, context);
      await invalidateInventory(queryClient);
    },
  });
};
