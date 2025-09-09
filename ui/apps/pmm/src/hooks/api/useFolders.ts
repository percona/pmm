import { useQuery, UseQueryOptions } from '@tanstack/react-query';
import { getDashboardFolders } from 'api/folders';
import { GetFoldersResponse } from 'types/folders.types';

export const useFolders = (
  options?: Partial<UseQueryOptions<GetFoldersResponse>>
) =>
  useQuery({
    queryKey: ['folders'],
    queryFn: async () => getDashboardFolders(),
    ...options,
  });
