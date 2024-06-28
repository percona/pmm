import { AxiosResponse } from 'axios';
import {
  GetUpdateStatusBody,
  GetUpdateStatusResponse,
  GetUpdatesBody,
  GetUpdatesResponse,
  StartUpdateBody,
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

export const startUpdate = async (body: StartUpdateBody) => {
  const res = await api.post<
    StartUpdateBody,
    AxiosResponse<StartUpdateResponse>
  >('/Updates/Start', body);
  return res.data;
};

export const getUpdateStatus = async (body: GetUpdateStatusBody) => {
  const res = await api.post<
    GetUpdateStatusBody,
    AxiosResponse<GetUpdateStatusResponse>
  >('/Updates/Status', body);
  return res.data;
};
