import { checkForUpdates, getChangeLogs, startUpdate } from 'api/updates';
import {
  useMutation,
  UseMutationOptions,
  useQuery,
  UseQueryOptions,
} from '@tanstack/react-query';
import {
  GetChangeLogsResponse,
  GetUpdatesResponse,
  StartUpdateBody,
  StartUpdateResponse,
} from 'types/updates.types';
import { AxiosError } from 'axios';

export const useCheckUpdates = (
  options?: UseQueryOptions<GetUpdatesResponse>
) =>
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
    ...options,
  });

export const useStartUpdate = (
  options?: UseMutationOptions<StartUpdateResponse, unknown, StartUpdateBody>
) =>
  useMutation({
    mutationFn: (args) => startUpdate(args),
    ...options,
  });

export const useChangeLogs = (
  options?: UseQueryOptions<GetChangeLogsResponse>
) =>
  useQuery({
    queryKey: ['changeLogs'],
    queryFn: getChangeLogs,
    ...options,
  });
