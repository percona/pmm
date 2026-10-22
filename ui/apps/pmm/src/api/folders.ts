import {
  CreateFolderPayload,
  DashboardFolder,
  GetFoldersResponse,
} from 'types/folders.types';
import { grafanaApi } from './api';

export const getDashboardFolders = async () => {
  const res = await grafanaApi.get<GetFoldersResponse>('/folders');
  return res.data;
};

export const createDashboardFolder = async (payload: CreateFolderPayload) => {
  const res = await grafanaApi.post<DashboardFolder>('/folders', payload);
  return res.data;
};
