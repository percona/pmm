import { useMemo } from 'react';
import { useAlertmanagerAlerts } from 'hooks/api/useAlertmanagerAlerts';
import { usePrometheusAlertRules } from 'hooks/api/usePrometheusAlertRules';
import { RDS_REFETCH_INTERVAL_MS } from 'hooks/api/useRds';
import { getFiringAlerts } from './AmazonRds.utils';

// The counts stay unknown until both requests succeed, so silenced alerts never count as firing;
// a failed or forbidden request leaves the rest of the page as it is.
export const useRdsAlerts = () => {
  const options = { refetchInterval: RDS_REFETCH_INTERVAL_MS };
  const { data: rules } = usePrometheusAlertRules(options);
  const { data: silencedAlerts, isSuccess: silencesLoaded } =
    useAlertmanagerAlerts(options);

  return useMemo(
    () =>
      rules && silencesLoaded
        ? getFiringAlerts(rules, silencedAlerts)
        : undefined,
    [rules, silencedAlerts, silencesLoaded]
  );
};
