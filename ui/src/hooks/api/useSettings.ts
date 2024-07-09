import { useQuery, UseQueryOptions } from '@tanstack/react-query';
import { getSettings } from 'api/settings';

export const useSettings = (options?: Partial<UseQueryOptions>) =>
  useQuery({
    queryKey: ['settings'],
    queryFn: () => getSettings(),
    ...options,
  });
