import { useQuery, UseQueryOptions } from '@tanstack/react-query';
import { getPrometheusAlertRules } from 'api/alerting';
import { PrometheusAlertRulesResponse } from 'types/alerting.types';

export const PROMETHEUS_ALERT_RULES_QUERY_KEY = ['alerting:prometheusRules'];

export const usePrometheusAlertRules = (
  options?: Partial<UseQueryOptions<PrometheusAlertRulesResponse>>
) =>
  useQuery({
    queryKey: PROMETHEUS_ALERT_RULES_QUERY_KEY,
    queryFn: () => getPrometheusAlertRules(),
    ...options,
  });
