import { useQuery, UseQueryOptions } from '@tanstack/react-query';
import { getAlertmanagerSilences } from 'api/alerting';
import { AlertmanagerSilence } from 'types/alerting.types';

export const ALERTMANAGER_SILENCES_QUERY_KEY = [
  'alerting:alertmanagerSilences',
];

export const useAlertmanagerSilences = (
  options?: Partial<UseQueryOptions<AlertmanagerSilence[]>>
) =>
  useQuery({
    queryKey: ALERTMANAGER_SILENCES_QUERY_KEY,
    queryFn: () => getAlertmanagerSilences(),
    ...options,
  });
