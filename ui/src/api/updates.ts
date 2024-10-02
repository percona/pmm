import { AxiosResponse } from 'axios';
import {
  GetChangeLogsResponse,
  GetUpdateStatusBody,
  GetUpdateStatusResponse,
  GetUpdatesParams,
  GetUpdatesResponse,
  StartUpdateBody,
  StartUpdateResponse,
} from 'types/updates.types';
import { api } from './api';

export const checkForUpdates = async (
  params: GetUpdatesParams = { force: false }
) => {
  const res = await api.get<GetUpdatesResponse>('/server/updates', {
    params,
  });
  return res.data;
};

export const startUpdate = async (body: StartUpdateBody) => {
  const res = await api.post<
    StartUpdateBody,
    AxiosResponse<StartUpdateResponse>
  >('/server/updates:start', body);
  return res.data;
};

export const getUpdateStatus = async (body: GetUpdateStatusBody) => {
  const res = await api.post<
    GetUpdateStatusBody,
    AxiosResponse<GetUpdateStatusResponse>
  >('/server/updates:getStatus', body);
  return res.data;
};

export const getChangeLogs = async (): Promise<GetChangeLogsResponse> => {
  const res = await api.get<GetChangeLogsResponse>(
    '/server/updates/changelogs'
  );
  return res.data;
};
