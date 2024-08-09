import { useQuery, UseQueryOptions } from '@tanstack/react-query';
import { getSettings } from 'api/settings';
import { Settings } from 'types/settings.types';

export const useSettings = (options?: Partial<UseQueryOptions<Settings>>) =>
  useQuery({
    queryKey: ['settings'],
    queryFn: () => getSettings(),
    ...options,
  });
