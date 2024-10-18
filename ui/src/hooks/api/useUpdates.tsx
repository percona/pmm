import { checkForUpdates, startUpdate } from 'api/updates';
import {
  useMutation,
  UseMutationOptions,
  useQuery,
} from '@tanstack/react-query';
import { StartUpdateBody, StartUpdateResponse } from 'types/updates.types';
import { AxiosError } from 'axios';

export const useCheckUpdates = () =>
  useQuery({
    queryKey: ['checkUpdates'],
    queryFn: async () => {
      try {
        return await checkForUpdates();
      } catch (error) {
        if ((error as AxiosError).response?.status !== 401) {
          return await checkForUpdates({
            force: false,
            onlyInstalledVersion: true,
          });
        }

        throw error;
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
