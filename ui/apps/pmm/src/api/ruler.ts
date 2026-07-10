import { GrafanaRulerRuleDTO } from 'types/ruler.types';
import { grafanaApi } from './api';

export const getRulerRule = async (uid: string) => {
  const res = await grafanaApi.get<GrafanaRulerRuleDTO>(
    `/ruler/grafana/api/v1/rule/${uid}`
  );
  return res.data;
};
