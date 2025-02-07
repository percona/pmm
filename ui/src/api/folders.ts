import { grafanaApi } from './api';
import { DashboardFolder } from 'types/folder.types';

export const getFolders = async () => {
  const res = await grafanaApi.get<DashboardFolder[]>('/folders');
  return res.data;
};
