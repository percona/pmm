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
import messenger from 'lib/messenger';
import {
  FrontendSettings,
  ReadonlySettings,
  Settings,
  UpdateSettingsPayload,
} from 'types/settings.types';

export const SETTINGS_QUERY_KEY = ['settings'] as const;

export const useSettings = (options?: Partial<UseQueryOptions<Settings>>) =>
  useQuery({
    queryKey: SETTINGS_QUERY_KEY,
    queryFn: () => getSettings(options?.axios),
    ...options,
  });

export const useReadonlySettings = (
  options?: Partial<UseQueryOptions<ReadonlySettings>>
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
  options?: Partial<UseMutationOptions<Settings, Error, UpdateSettingsPayload>>
) => {
  const queryClient = useQueryClient();
  const settings = useSettings({
    enabled: false,
    retry: 24,
    retryDelay: 2500,
    axios: {
      disableNotifications: true,
    },
  });

  return useMutation({
    mutationFn: async (payload) => {
      const prevAddress =
        queryClient.getQueryData<Settings>(
          SETTINGS_QUERY_KEY
        )?.pmmPublicAddress;
      const data = await updateSettings(payload);

      // nginx is getting reset when public address is changing
      // so we need to make sure the UI is accessible after the update
      if (prevAddress !== data.pmmPublicAddress) {
        await new Promise((resolve) => setTimeout(resolve, 2500));
        await settings.refetch({ throwOnError: true });
      }

      return data;
    },
    ...options,
    onSuccess: (data, variables, onMutate, context) => {
      void queryClient.invalidateQueries({ queryKey: SETTINGS_QUERY_KEY });
      messenger.sendMessage({
        type: 'SETTINGS_CHANGED',
      });
      options?.onSuccess?.(data, variables, onMutate, context);
    },
  });
};
