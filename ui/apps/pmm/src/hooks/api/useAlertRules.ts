import { queryOptions, useQuery, UseQueryOptions } from '@tanstack/react-query';
import { getAlertRules } from 'api/alerting';

const alertRulesOptions = (options?: UseQueryOptions) =>
  queryOptions({
    queryKey: ['alerting:rules'],
    queryFn: async () => {
      const data = await getAlertRules();
      return data;
    },
    ...options,
  });

export const useAlertRules = (options?: UseQueryOptions) =>
  useQuery(alertRulesOptions(options));
