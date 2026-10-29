import type { StarredDashboard } from 'types/dashboards.types';
import { grafanaApi } from './api';

// The signed-in user's starred dashboards, as Grafana keeps them.
export const getStarredDashboards = async () => {
  const res = await grafanaApi.get<StarredDashboard[]>('/search', {
    params: { starred: true, type: 'dash-db' },
  });
  return res.data;
};
