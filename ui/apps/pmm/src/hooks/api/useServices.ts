import {
  type UseMutationOptions,
  type UseQueryOptions,
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query';
import {
  addAzureService,
  addService,
  changeService,
  discoverAzure,
  discoverRDS,
  getService,
  getServiceTypes,
  listServices,
  listManagedServices,
  removeService,
} from 'api/services';
import type {
  AddAzureDatabasePayload,
  AddServicePayload,
  BaseService,
  ChangeServiceParams,
  DiscoverAzurePayload,
  DiscoverAzureResponse,
  DiscoverRDSPayload,
  DiscoverRDSResponse,
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

// The inventory refreshes in the background, so the caller moves on without waiting for it.
const useAddMutation = <T>(
  mutationKey: string[],
  mutationFn: (payload: T) => Promise<void>,
  options?: Partial<UseMutationOptions<void, Error, T>>
) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationKey,
    mutationFn,
    ...options,
    onSuccess: (data, variables, onMutate, context) => {
      void invalidateInventory(queryClient);
      return options?.onSuccess?.(data, variables, onMutate, context);
    },
  });
};

export const useAddService = (
  options?: Partial<UseMutationOptions<void, Error, AddServicePayload>>
) => useAddMutation(['services:add'], addService, options);

export const useAddAzureService = (
  options?: Partial<UseMutationOptions<void, Error, AddAzureDatabasePayload>>
) => useAddMutation(['services:add-azure'], addAzureService, options);

// Discovery with PMM Server's own AWS identity; aborted when the page is left.
export const useServerRDSInstances = (
  options?: Partial<UseQueryOptions<DiscoverRDSResponse>>
) =>
  useQuery({
    queryKey: ['services:discover-rds-server'],
    queryFn: ({ signal }) => discoverRDS({}, signal),
    ...options,
  });

export const useDiscoverRDS = (
  options?: Partial<
    UseMutationOptions<DiscoverRDSResponse, Error, DiscoverRDSPayload>
  >
) =>
  useMutation({
    mutationKey: ['services:discover-rds'],
    mutationFn: (payload) => discoverRDS(payload),
    ...options,
  });

export const useDiscoverAzure = (
  options?: Partial<
    UseMutationOptions<DiscoverAzureResponse, Error, DiscoverAzurePayload>
  >
) =>
  useMutation({
    mutationKey: ['services:discover-azure'],
    mutationFn: discoverAzure,
    ...options,
  });

export const useChangeService = (
  options?: Partial<UseMutationOptions<void, Error, ChangeServiceParams>>
) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (params: ChangeServiceParams) => changeService(params),
    ...options,
    onSuccess: async (data, variables, onMutate, context) => {
      await options?.onSuccess?.(data, variables, onMutate, context);
      await invalidateInventory(queryClient);
    },
  });
};
