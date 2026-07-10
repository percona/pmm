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
import { api } from './api';

export const listDumps = async (): Promise<ListDumpsResponse> => {
  const response = await api.get<ListDumpsResponse>('/dumps');
  return response.data;
};

export const startDump = async (
  payload: StartDumpPayload
): Promise<StartDumpResponse> => {
  const response = await api.post<StartDumpResponse>('/dumps:start', payload);
  return response.data;
};

export const deleteDumps = async (
  payload: DeleteDumpsPayload
): Promise<EmptyResponse> => {
  const response = await api.post<EmptyResponse>('/dumps:batchDelete', payload);
  return response.data;
};

export const getDumpLogs = async ({
  dumpId,
  offset,
  limit,
}: GetDumpLogsParams): Promise<GetDumpLogsResponse> => {
  const response = await api.get<GetDumpLogsResponse>(`/dumps/${dumpId}/logs`, {
    params: { offset, limit },
  });
  return response.data;
};

export const uploadDumps = async (
  payload: UploadDumpsPayload
): Promise<EmptyResponse> => {
  const response = await api.post<EmptyResponse>('/dumps:upload', payload);
  return response.data;
};
