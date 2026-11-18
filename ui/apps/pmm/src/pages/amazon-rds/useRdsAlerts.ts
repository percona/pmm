import { useMemo } from 'react';
import { useAlertmanagerAlerts } from 'hooks/api/useAlertmanagerAlerts';
import { usePrometheusAlertRules } from 'hooks/api/usePrometheusAlertRules';
import { RDS_REFETCH_INTERVAL_MS } from 'hooks/api/useRds';
import { getFiringAlerts } from './AmazonRds.utils';

// The counts stay unknown unless both requests succeeded on their last try, so silenced alerts never
// count as firing and a failed refresh never shows stale counts; the rest of the page stays as it is.
export const useRdsAlerts = () => {
  const options = { refetchInterval: RDS_REFETCH_INTERVAL_MS };
  const { data: rules, isSuccess: rulesLoaded } =
    usePrometheusAlertRules(options);
  const { data: silencedAlerts, isSuccess: silencesLoaded } =
    useAlertmanagerAlerts(options);

  return useMemo(
    () =>
      rules && rulesLoaded && silencesLoaded
        ? getFiringAlerts(rules, silencedAlerts)
        : undefined,
    [rules, rulesLoaded, silencedAlerts, silencesLoaded]
  );
};
