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

export type UseCheckUpdatesOptions = Partial<
  UseQueryOptions<GetUpdatesResponse>
> & {
  // deployments with PMM_ENABLE_UPDATES=0 reject a full check, so ask only for
  // what the server always answers
  onlyInstalledVersion?: boolean;
};

export const useCheckUpdates = ({
  onlyInstalledVersion = false,
  ...options
}: UseCheckUpdatesOptions = {}) =>
  useQuery({
    queryKey: ['checkUpdates', onlyInstalledVersion],
    queryFn: async () => {
      if (onlyInstalledVersion) {
        return checkForUpdates({ force: false, onlyInstalledVersion: true });
      }

      try {
        // 400 is "updates are disabled", which the fallback below handles. Any
        // other failure is real and the user needs to hear about it.
        return await checkForUpdates(
          { force: true },
          { disableNotifications: (error) => error.response?.status === 400 }
        );
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
