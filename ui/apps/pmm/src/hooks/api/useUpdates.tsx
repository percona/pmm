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
import { ApiError } from 'types/api.types';

export const useCheckUpdates = (
  options?: Partial<UseQueryOptions<GetUpdatesResponse>>
) =>
  useQuery({
    queryKey: ['checkUpdates'],
    queryFn: async () => {
      try {
        return await checkForUpdates({ force: true });
      } catch (error) {
        if ((error as AxiosError).response?.status !== 401) {
          return await checkForUpdates({
            force: true,
            onlyInstalledVersion: true,
          });
        }

        throw error;
      }
    },
    ...options,
  });

export const useStartUpdate = (
  options?: UseMutationOptions<StartUpdateResponse, ApiError, StartUpdateBody>
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
