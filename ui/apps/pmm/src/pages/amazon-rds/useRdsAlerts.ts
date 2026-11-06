import { useMemo } from 'react';
import { useAlertmanagerAlerts } from 'hooks/api/useAlertmanagerAlerts';
import { usePrometheusAlertRules } from 'hooks/api/usePrometheusAlertRules';
import { RDS_REFETCH_INTERVAL_MS } from 'hooks/api/useRds';
import { getFiringAlerts } from './AmazonRds.utils';

// A failed or forbidden request leaves the alert counts unknown and the rest of the page as it is.
export const useRdsAlerts = () => {
  const options = { refetchInterval: RDS_REFETCH_INTERVAL_MS };
  const { data: rules } = usePrometheusAlertRules(options);
  const { data: silencedAlerts } = useAlertmanagerAlerts(options);

  return useMemo(
    () => rules && getFiringAlerts(rules, silencedAlerts),
    [rules, silencedAlerts]
  );
};
