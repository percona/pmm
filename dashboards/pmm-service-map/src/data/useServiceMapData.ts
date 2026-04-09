import { useEffect, useState } from 'react';
import { getDataSourceSrv } from '@grafana/runtime';
import { DataFrame, type DataSourceApi, DataQueryRequest, TimeRange } from '@grafana/data';
import { ServiceMapData, ServiceMapOptions } from '../types';
import { transformToServiceMap, buildIpToAppIdMap } from './transform';

const QUERIES = [
  { refId: 'requests', expr: 'sum by (app_id, destination, actual_destination, proto, status) (rr_connection_l7_requests)' },
  { refId: 'latency', expr: 'sum by (app_id, destination, actual_destination, proto) (rr_connection_l7_latency)' },
  { refId: 'bytesSent', expr: 'sum by (app_id, destination, actual_destination) (rr_connection_tcp_bytes_sent)' },
  { refId: 'bytesRecv', expr: 'sum by (app_id, destination, actual_destination) (rr_connection_tcp_bytes_received)' },
  { refId: 'tcpFailed', expr: 'sum by (app_id, destination, actual_destination) (rr_connection_tcp_failed)' },
  { refId: 'listenInfo', expr: 'container_net_tcp_listen_info' },
];

async function runPromQuery(
  ds: DataSourceApi,
  expr: string,
  refId: string,
  range: TimeRange
): Promise<DataFrame[]> {
  const request = {
    targets: [{ refId, expr, instant: true, range: false, format: 'time_series' }],
    range,
    intervalMs: 60000,
    maxDataPoints: 1,
    requestId: `svcmap-${refId}`,
  } as unknown as DataQueryRequest;

  return new Promise((resolve, reject) => {
    (ds as any).query(request).subscribe({
      next: (response: { data: DataFrame[] }) => resolve(response.data ?? []),
      error: (err: unknown) => reject(err),
    });
  });
}

export function useServiceMapData(
  options: ServiceMapOptions,
  timeRange: TimeRange
): { data: ServiceMapData | null; loading: boolean; error: string | null } {
  const [data, setData] = useState<ServiceMapData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function fetchData() {
      setLoading(true);
      setError(null);

      try {
        const dsSrv = getDataSourceSrv();
        const dsName = options.promDatasource || undefined;
        const ds = await dsSrv.get(dsName);

        const results = await Promise.all(
          QUERIES.map((q) => runPromQuery(ds, q.expr, q.refId, timeRange))
        );

        if (cancelled) {
          return;
        }

        const ipMap = buildIpToAppIdMap(results[5]);

        const mapData = transformToServiceMap(
          results[0], // requests
          results[1], // latency
          results[2], // bytesSent
          results[3], // bytesRecv
          results[4], // tcpFailed
          ipMap,
          options
        );

        setData(mapData);
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : String(err));
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    fetchData();
    return () => {
      cancelled = true;
    };
  }, [options.promDatasource, options.errorAmberThreshold, options.errorRedThreshold, options.minEdgeWeight, timeRange]);

  return { data, loading, error };
}
