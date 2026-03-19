import {
  useMutation,
  UseMutationOptions,
  useQuery,
  useQueryClient,
  UseQueryOptions,
} from '@tanstack/react-query';
import {
  getFrontendSettings,
  getReadonlySettings,
  getSettings,
  updateSettings,
} from 'api/settings';
import {
  FrontendSettings,
  Settings,
  UpdateSettingsPayload,
} from 'types/settings.types';

export const SETTINGS_QUERY_KEY = ['settings'] as const;

export const useSettings = (options?: Partial<UseQueryOptions<Settings>>) =>
  useQuery({
    queryKey: SETTINGS_QUERY_KEY,
    queryFn: () => getSettings(),
    ...options,
  });

export const useReadonlySettings = (
  options?: Partial<UseQueryOptions<Settings>>
) =>
  useQuery({
    queryKey: ['settings:readonly'],
    queryFn: () => getReadonlySettings(),
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

export const useUpdateSettings = (
  options?: Partial<
    UseMutationOptions<Settings, Error, UpdateSettingsPayload>
  >
) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (payload) => updateSettings(payload),
    onSuccess: async (data, variables, context) => {
      await queryClient.invalidateQueries({ queryKey: SETTINGS_QUERY_KEY });
      await options?.onSuccess?.(data, variables, context);
    },
    ...options,
  });
};
