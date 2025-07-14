import { useQuery, UseQueryOptions } from '@tanstack/react-query';
import { getFrontendSettings, getSettings } from 'api/settings';
import { FrontendSettings, Settings } from 'types/settings.types';

export const useSettings = (options?: Partial<UseQueryOptions<Settings>>) =>
  useQuery({
    queryKey: ['settings'],
    queryFn: () => getSettings(),
    ...options,
  });

export const useFrontendSettings = (
  options?: Partial<UseQueryOptions<FrontendSettings>>
) =>
  useQuery({
    queryKey: ['frontendSettings'],
    queryFn: () => getFrontendSettings(),
    ...options,
  });
