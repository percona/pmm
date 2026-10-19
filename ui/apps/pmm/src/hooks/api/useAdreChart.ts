import { useQuery } from '@tanstack/react-query';
import { useMemo } from 'react';
import {
  getGrafanaDatasources,
  queryGrafanaDatasource,
} from 'api/grafanaQuery';
import {
  type ChartSpec,
  framesToSeries,
  rangeMs,
} from 'components/adre/adreChart.utils';

const MAX_POINTS = 300;
const MAX_SERIES = 10;

export const useGrafanaDatasources = () =>
  useQuery({
    queryKey: ['grafana', 'datasources'],
    queryFn: getGrafanaDatasources,
    staleTime: Infinity,
  });

/** The chart's series; a relative window is fixed when the chart first shows. */
export const useAdreChart = (spec: ChartSpec) => {
  const range = useMemo(
    () => rangeMs(spec.from, spec.to),
    [spec.from, spec.to]
  );
  const datasources = useGrafanaDatasources();
  const ds = datasources.data?.[spec.ds];
  const data = useQuery({
    queryKey: ['adre-chart', spec.ds, spec.query, range],
    enabled: !!ds && !!range,
    queryFn: async () =>
      framesToSeries(
        await queryGrafanaDatasource(ds!, spec.query, range!, MAX_POINTS),
        MAX_SERIES
      ),
    staleTime: 5 * 60_000,
    retry: false,
  });

  let error: string | undefined;
  if (!range) error = `Can't read the time range ${spec.from} – ${spec.to}`;
  else if (datasources.isSuccess && !ds)
    error = `No datasource named ${spec.ds}`;
  else if (datasources.error) error = 'Could not load the datasources';
  else if (data.error) error = data.error.message;

  return {
    range,
    ds,
    error,
    loading: !error && (datasources.isLoading || data.isLoading),
    data: data.data,
  };
};
