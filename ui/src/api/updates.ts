import { AxiosResponse } from 'axios';
import {
  GetUpdateStatusBody,
  GetUpdateStatusResponse,
  GetUpdatesBody,
  GetUpdatesResponse,
  StartUpdateResponse,
} from 'types/updates.types';
import { api } from './api';

export const checkForUpdates = async (
  body: GetUpdatesBody = { force: false }
) => {
  const res = await api.post<GetUpdatesBody, AxiosResponse<GetUpdatesResponse>>(
    '/Updates/Check',
    body
  );
  return res.data;
};

export const startUpdate = async () => {
  const res = await api.post<object, AxiosResponse<StartUpdateResponse>>(
    '/Updates/Start',
    {}
  );
  return res.data;
};

export const getUpdateStatus = async (body: GetUpdateStatusBody) => {
  const res = await api.post<
    GetUpdateStatusBody,
    AxiosResponse<GetUpdateStatusResponse>
  >('/Updates/Status', body);
  return res.data;
};
