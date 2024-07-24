import { checkForUpdates, startUpdate } from 'api/updates';
import {
  useMutation,
  UseMutationOptions,
  useQuery,
} from '@tanstack/react-query';
import { AxiosError } from 'axios';
import { StartUpdateBody, StartUpdateResponse } from 'types/updates.types';
import { ApiErrorResponse } from 'types/api.types';

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
    mutationFn: async (args) => {
      try {
        return await startUpdate(args);
      } catch (error) {
        const { response } = error as AxiosError<ApiErrorResponse>;

        if (response?.status === 499 || response?.data?.code === 14) {
          return;
        }

        throw error;
      }
    },
    ...options,
  });
