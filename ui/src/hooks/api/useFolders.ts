import { useQuery } from '@tanstack/react-query';
import { getFolders } from 'api/folders';

export const useDashboardFolders = () =>
  useQuery({
    queryKey: ['dashboard-folders'],
    queryFn: getFolders,
  });
