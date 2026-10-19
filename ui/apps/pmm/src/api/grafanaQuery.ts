import { grafanaApi } from './api';
import type { DataFrameJSON } from 'components/adre/adreChart.utils';

export interface GrafanaDatasourceRef {
  uid: string;
  type: string;
}

/** The datasources the current user can query, by name, as Grafana's own frontend loads them. */
export const getGrafanaDatasources = async () => {
  const res = await grafanaApi.get<{
    datasources: Record<string, GrafanaDatasourceRef>;
  }>('/frontend/settings');
  return res.data.datasources;
};

/** The query fields each datasource type takes; ClickHouse format 0 returns a time series. */
export const datasourceQuery = (type: string, query: string) => {
  if (type === 'prometheus') return { expr: query, range: true };
  if (type === 'grafana-clickhouse-datasource')
    return { rawSql: query, format: 0, editorType: 'sql' };
  return undefined;
};

/** Runs one query through Grafana as the current user and returns its data frames. */
export const queryGrafanaDatasource = async (
  ds: GrafanaDatasourceRef,
  query: string,
  range: { from: number; to: number },
  maxDataPoints: number
) => {
  const fields = datasourceQuery(ds.type, query);
  if (!fields) throw new Error(`Charts can't query ${ds.type} datasources`);
  const res = await grafanaApi.post<{
    results: Record<string, { frames?: DataFrameJSON[]; error?: string }>;
  }>('/ds/query', {
    from: String(range.from),
    to: String(range.to),
    queries: [
      {
        refId: 'A',
        datasource: ds,
        maxDataPoints,
        intervalMs: Math.max(
          15_000,
          Math.round((range.to - range.from) / maxDataPoints)
        ),
        ...fields,
      },
    ],
  });
  const result = res.data.results.A;
  if (result?.error) throw new Error(result.error);
  return result?.frames ?? [];
};
