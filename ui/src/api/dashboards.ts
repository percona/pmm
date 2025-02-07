import { grafanaApi } from './api';

export const starDashboard = async (uid: string) => {
  const res = await grafanaApi.post('/dashboard/uid/' + uid);
  return res.data;
};

export const unstarDashboard = async (uid: string) => {
  const res = await grafanaApi.delete('/dashboard/uid/' + uid);
  res.data;
};
