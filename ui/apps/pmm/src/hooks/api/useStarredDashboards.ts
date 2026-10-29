import { useQuery, type UseQueryOptions } from '@tanstack/react-query';
import { getStarredDashboards } from 'api/dashboards';
import type { StarredDashboard } from 'types/dashboards.types';

export const STARRED_DASHBOARDS_QUERY_KEY = ['dashboards:starred'];

export const useStarredDashboards = (
  options?: Partial<UseQueryOptions<StarredDashboard[]>>
) =>
  useQuery({
    queryKey: STARRED_DASHBOARDS_QUERY_KEY,
    queryFn: () => getStarredDashboards(),
    ...options,
  });
