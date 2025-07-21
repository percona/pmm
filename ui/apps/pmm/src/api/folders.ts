import { GetFoldersResponse } from 'types/folders.types';
import { grafanaApi } from './api';

export const getDashboardFolders = async () => {
  const res = await grafanaApi.get<GetFoldersResponse>('/folders');
  return res.data;
};
