import { queryOptions, useQuery, UseQueryOptions } from '@tanstack/react-query';
import { getPrometheusAlertRules } from 'api/alerting';
import { PrometheusAlertRulesResponse } from 'types/alerting.types';

export const PROMETHEUS_ALERT_RULES_QUERY_KEY = ['alerting:prometheusRules'];

export const prometheusAlertsOptions = (
  options?: Partial<UseQueryOptions<PrometheusAlertRulesResponse>>
) =>
  queryOptions({
    queryKey: PROMETHEUS_ALERT_RULES_QUERY_KEY,
    queryFn: () => getPrometheusAlertRules(),
    ...options,
  });

export const usePrometheusAlertRules = (
  options?: Partial<UseQueryOptions<PrometheusAlertRulesResponse>>
) => useQuery(prometheusAlertsOptions(options));
