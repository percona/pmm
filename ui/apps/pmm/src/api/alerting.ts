import { PrometheusAlertRulesResponse } from 'types/alerting.types';
import { grafanaApi } from './api';

export const getPrometheusAlertRules = async () => {
  const response = await grafanaApi.get<PrometheusAlertRulesResponse>(
    '/prometheus/grafana/api/v1/rules'
  );
  return response.data;
};
