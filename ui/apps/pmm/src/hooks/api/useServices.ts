import {
  type QueryClient,
  useMutation,
  type UseMutationOptions,
  useQuery,
  useQueryClient,
  type UseQueryOptions,
} from '@tanstack/react-query';
import {
  addAzureService,
  addService,
  discoverAzure,
  discoverRDS,
  getServiceTypes,
  listServices,
  listManagedServices,
} from 'api/services';
import { NODES_QUERY_KEY } from './useNodes';
import type {
  AddAzureDatabasePayload,
  AddServicePayload,
  DiscoverAzurePayload,
  DiscoverAzureResponse,
  DiscoverRDSPayload,
  DiscoverRDSResponse,
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

const invalidateInventory = (queryClient: QueryClient) =>
  Promise.all(
    [
      ['services:getTypes'],
      ['services:list'],
      ['services:managed'],
      NODES_QUERY_KEY,
    ].map((queryKey) => queryClient.invalidateQueries({ queryKey }))
  );

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
