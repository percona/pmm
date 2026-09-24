import { useQuery, UseQueryOptions } from '@tanstack/react-query';
import { getAlertmanagerAlerts } from 'api/alerting';
import { AlertmanagerAlert } from 'types/alerting.types';

export const ALERTMANAGER_ALERTS_QUERY_KEY = ['alerting:alertmanagerAlerts'];

export const useAlertmanagerAlerts = (
  options?: Partial<UseQueryOptions<AlertmanagerAlert[]>>
) =>
  useQuery({
    queryKey: ALERTMANAGER_ALERTS_QUERY_KEY,
    queryFn: () => getAlertmanagerAlerts(),
    ...options,
  });
