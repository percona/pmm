import {
  AlertEvalResponse,
  GrafanaAlertQuery,
  GrafanaAlertRuleDefinition,
  PrometheusAlertRulesResponse,
} from 'types/alerting.types';
import { grafanaApi } from './api';

export const getPrometheusAlertRules = async () => {
  const response = await grafanaApi.get<PrometheusAlertRulesResponse>(
    '/prometheus/grafana/api/v1/rules'
  );
  return response.data;
};

export const getAlertRuleDefinition = async (uid: string) => {
  const response = await grafanaApi.get<GrafanaAlertRuleDefinition>(
    `/v1/provisioning/alert-rules/${uid}`
  );
  return response.data;
};

export const evalAlertQueries = async (data: GrafanaAlertQuery[]) => {
  const response = await grafanaApi.post<AlertEvalResponse>('/v1/eval', {
    data,
  });
  return response.data;
};
