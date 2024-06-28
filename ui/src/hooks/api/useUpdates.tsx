import { checkForUpdates, startUpdate, StartUpdateBody } from 'api/updates';
import {
  useMutation,
  UseMutationOptions,
  useQuery,
} from '@tanstack/react-query';

export const useCheckUpdates = () =>
  useQuery({
    queryKey: ['checkUpdates'],
    queryFn: async () => {
      try {
        return await checkForUpdates();
      } catch (error) {
        return await checkForUpdates({
          force: false,
          onlyInstalledVersion: true,
        });
      }
    },
  });

export const useStartUpdate = (
  options?: UseMutationOptions<unknown, unknown, StartUpdateBody>
) =>
  useMutation({
    mutationFn: (args) => startUpdate(args),
    ...options,
  });
