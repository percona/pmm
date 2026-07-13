import {
  useMutation,
  UseMutationOptions,
  useQuery,
  useQueryClient,
  UseQueryOptions,
} from '@tanstack/react-query';
import {
  deleteDumps,
  getDumpLogs,
  listDumps,
  startDump,
  uploadDumps,
} from 'api/dump';
import {
  DeleteDumpsPayload,
  GetDumpLogsParams,
  GetDumpLogsResponse,
  ListDumpsResponse,
  StartDumpPayload,
  StartDumpResponse,
  UploadDumpsPayload,
} from 'types/dump.types';
import { EmptyResponse } from 'types/util.types';

export const DUMP_QUERY_KEYS = {
  list: ['dumps:list'] as const,
  logs: (dumpId: string) => ['dumps:logs', dumpId] as const,
  start: ['dumps:start'] as const,
  delete: ['dumps:delete'] as const,
  upload: ['dumps:upload'] as const,
};

export const useDumps = (
  options?: Partial<UseQueryOptions<ListDumpsResponse>>
) =>
  useQuery({
    queryKey: DUMP_QUERY_KEYS.list,
    queryFn: listDumps,
    refetchInterval: 5_000,
    ...options,
  });

export const useStartDump = (
  options?: Partial<
    UseMutationOptions<StartDumpResponse, Error, StartDumpPayload>
  >
) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationKey: DUMP_QUERY_KEYS.start,
    mutationFn: startDump,
    ...options,
    onSuccess: async (data, variables, onMutateResult, context) => {
      await options?.onSuccess?.(data, variables, onMutateResult, context);
      await queryClient.invalidateQueries({ queryKey: DUMP_QUERY_KEYS.list });
    },
  });
};

export const useDeleteDumps = (
  options?: Partial<
    UseMutationOptions<EmptyResponse, Error, DeleteDumpsPayload>
  >
) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationKey: DUMP_QUERY_KEYS.delete,
    mutationFn: deleteDumps,
    ...options,
    onSuccess: async (data, variables, onMutateResult, context) => {
      await options?.onSuccess?.(data, variables, onMutateResult, context);
      await queryClient.invalidateQueries({ queryKey: DUMP_QUERY_KEYS.list });
    },
  });
};

export const useDumpLogs = (
  params: GetDumpLogsParams,
  options?: Partial<UseQueryOptions<GetDumpLogsResponse>>
) =>
  useQuery({
    queryKey: [...DUMP_QUERY_KEYS.logs(params.dumpId), params.offset],
    queryFn: () => getDumpLogs(params),
    enabled: !!params.dumpId,
    ...options,
  });

export const useUploadDumps = (
  options?: Partial<
    UseMutationOptions<EmptyResponse, Error, UploadDumpsPayload>
  >
) =>
  useMutation({
    mutationKey: DUMP_QUERY_KEYS.upload,
    mutationFn: uploadDumps,
    ...options,
  });
