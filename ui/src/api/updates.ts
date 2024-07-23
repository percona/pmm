import { AxiosResponse } from 'axios';
import {
  GetUpdateStatusBody,
  GetUpdateStatusResponse,
  GetUpdatesParams,
  GetUpdatesResponse,
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

export const startUpdate = async () => {
  const res = await api.post<object, AxiosResponse<StartUpdateResponse>>(
    '/server/updates:start',
    {}
  );
  return res.data;
};

export const getUpdateStatus = async (body: GetUpdateStatusBody) => {
  const res = await api.post<
    GetUpdateStatusBody,
    AxiosResponse<GetUpdateStatusResponse>
  >('/server/updates:getStatus', body);
  return res.data;
};
