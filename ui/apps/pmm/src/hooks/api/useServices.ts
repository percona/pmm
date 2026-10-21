import {
  type UseMutationOptions,
  type UseQueryOptions,
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query';
import {
  getEditableService,
  getService,
  getServiceTypes,
  listServices,
  listManagedServices,
  removeService,
  updateService,
} from 'api/services';
import type {
  BaseService,
  EditableService,
  ListServicesParams,
  ListServicesResponse,
  ListTypesResponse,
  ManagedServicesResponse,
  RemoveServiceParams,
  UpdateServiceParams,
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

export const useService = (
  serviceId?: string,
  options?: Partial<UseQueryOptions<BaseService | null>>
) =>
  useQuery({
    queryKey: ['services:get', serviceId],
    queryFn: () => getService(serviceId as string),
    enabled: !!serviceId,
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

export const useEditableService = (
  serviceId?: string,
  options?: Partial<UseQueryOptions<EditableService | null>>
) =>
  useQuery({
    queryKey: ['services:editable', serviceId],
    queryFn: () => getEditableService(serviceId as string),
    enabled: !!serviceId,
    ...options,
  });

export const useUpdateService = (
  options?: Partial<UseMutationOptions<string, Error, UpdateServiceParams>>
) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (params: UpdateServiceParams) => updateService(params),
    ...options,
    onSuccess: async (data, variables, onMutate, context) => {
      await options?.onSuccess?.(data, variables, onMutate, context);
      await invalidateInventory(queryClient);
    },
  });
};
