import { getCurrentVersion, startUpdate } from 'api/version';
import {
  UseMutationOptions,
  useMutation,
  useQuery,
} from '@tanstack/react-query';
import { StartUpdateBody } from 'types/version.types';

export const useCheckUpdates = () =>
  useQuery({
    queryKey: ['checkUpdates'],
    queryFn: async () => {
      try {
        return await getCurrentVersion();
      } catch (error) {
        return await getCurrentVersion({
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
