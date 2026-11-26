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
import { ApiError, ApiErrorResponse } from 'types/api.types';

// grpc-gateway maps several gRPC codes onto HTTP 400, so the status alone cannot
// tell "updates are disabled" apart from a rejected request.
const GRPC_FAILED_PRECONDITION = 9;

const isUpdatesDisabled = (error: AxiosError) =>
  error.response?.status === 400 &&
  (error.response.data as ApiErrorResponse | undefined)?.code ===
    GRPC_FAILED_PRECONDITION;

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
        // the fallback below handles a disabled deployment. Any other failure
        // is real and the user needs to hear about it.
        return await checkForUpdates(
          { force: true },
          { disableNotifications: isUpdatesDisabled }
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
