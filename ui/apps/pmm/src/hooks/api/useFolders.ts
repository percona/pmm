import {
  useMutation,
  UseMutationOptions,
  useQuery,
  useQueryClient,
  UseQueryOptions,
} from '@tanstack/react-query';
import { createDashboardFolder, getDashboardFolders } from 'api/folders';
import {
  CreateFolderPayload,
  DashboardFolder,
  GetFoldersResponse,
} from 'types/folders.types';

export const useFolders = (
  options?: Partial<UseQueryOptions<GetFoldersResponse>>
) =>
  useQuery({
    queryKey: ['folders'],
    queryFn: async () => getDashboardFolders(),
    ...options,
  });

export const useCreateFolder = (
  options?: Partial<
    UseMutationOptions<DashboardFolder, Error, CreateFolderPayload>
  >
) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationKey: ['folders:create'],
    mutationFn: createDashboardFolder,
    ...options,
    onSuccess: async (data, variables, onMutate, context) => {
      await options?.onSuccess?.(data, variables, onMutate, context);
      await queryClient.invalidateQueries({ queryKey: ['folders'] });
    },
  });
};
