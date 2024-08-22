import { checkForUpdates, getChangelogs, startUpdate } from 'api/updates';
import {
  useMutation,
  UseMutationOptions,
  useQuery,
  UseQueryOptions,
} from '@tanstack/react-query';
import {
  GetChangelogsResponse,
  StartUpdateBody,
  StartUpdateResponse,
} from 'types/updates.types';
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
  options?: UseMutationOptions<StartUpdateResponse, unknown, StartUpdateBody>
) =>
  useMutation({
    mutationFn: (args) => startUpdate(args),
    ...options,
  });

export const useChangelogs = (
  options?: UseQueryOptions<GetChangelogsResponse>
) =>
  useQuery({
    queryKey: ['changelogs'],
    queryFn: getChangelogs,
    ...options,
  });
