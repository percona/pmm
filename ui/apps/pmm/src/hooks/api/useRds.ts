import { useQuery, type UseQueryOptions } from '@tanstack/react-query';
import { getRdsInstances } from 'api/rds';
import { useSettings } from 'contexts/settings';
import { METRICS_DATASOURCE } from 'lib/constants';
import type { RdsInstanceRow } from 'types/rds.types';

export const RDS_REFETCH_INTERVAL_MS = 60_000;

// Waits for Grafana's frontend settings, which hold the UID of the metrics data source.
export const useRdsInstances = (
  options?: Partial<UseQueryOptions<RdsInstanceRow[]>>
) => {
  const { settings } = useSettings();
  const datasourceUid =
    settings?.frontend.datasources?.[METRICS_DATASOURCE]?.uid;

  return useQuery({
    queryKey: ['rds:instances', datasourceUid],
    queryFn: () => {
      if (!datasourceUid) {
        throw new Error(`Data source '${METRICS_DATASOURCE}' not found`);
      }

      return getRdsInstances(datasourceUid);
    },
    enabled: !!settings,
    refetchInterval: RDS_REFETCH_INTERVAL_MS,
    ...options,
  });
};
