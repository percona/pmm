import { checkForUpdates, startUpdate } from 'api/updates';
import {
  useMutation,
  UseMutationOptions,
  useQuery,
} from '@tanstack/react-query';
import { StartUpdateBody, StartUpdateResponse } from 'types/updates.types';

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
  options?: UseMutationOptions<
    StartUpdateResponse | undefined,
    unknown,
    StartUpdateBody
  >
) =>
  useMutation({
    mutationFn: (args) => startUpdate(args),
    ...options,
  });
