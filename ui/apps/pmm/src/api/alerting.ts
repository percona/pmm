import {
  AlertEvalResponse,
  GrafanaAlertQuery,
  GrafanaRulerRuleDTO,
  PrometheusAlertRulesResponse,
} from 'types/alerting.types';
import { grafanaApi } from './api';

export const getPrometheusAlertRules = async () => {
  const response = await grafanaApi.get<PrometheusAlertRulesResponse>(
    '/prometheus/grafana/api/v1/rules'
  );
  return response.data;
};

export const evalAlertQueries = async (data: GrafanaAlertQuery[]) => {
  const response = await grafanaApi.post<AlertEvalResponse>('/v1/eval', {
    data,
  });
  return response.data;
};

export const getRulerRule = async (uid: string) => {
  const res = await grafanaApi.get<GrafanaRulerRuleDTO>(
    `/ruler/grafana/api/v1/rule/${uid}`
  );
  return res.data;
};
