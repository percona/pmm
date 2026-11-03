import { getBackendSrv, type BackendSrvRequest } from '@grafana/runtime';
import { API_BASE_URL } from './api.constants';

const OPTIONS: Partial<BackendSrvRequest> = { showErrorAlert: false };

export const get = <T>(url: string, params?: BackendSrvRequest['params']) =>
  getBackendSrv().get<T>(`${API_BASE_URL}/${url}`, params, undefined, OPTIONS);

export const post = <T, B = unknown>(url: string, data: B) =>
  getBackendSrv().post<T>(`${API_BASE_URL}/${url}`, data, OPTIONS);
