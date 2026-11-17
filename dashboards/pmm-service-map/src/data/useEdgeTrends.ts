import { useEffect, useState } from 'react';
import { getDataSourceSrv } from '@grafana/runtime';
import { DataFrame, type DataSourceApi, DataQueryRequest, FieldType, TimeRange } from '@grafana/data';
import { SelectedEdge } from '../types';

export interface TrendPoint {
  t: number;
  v: number;
}

function escapeProm(s: string): string {
  return s.replace(/\\/g, '\\\\').replace(/"/g, '\\"');
}

async function runRangeQuery(
  ds: DataSourceApi,
  expr: string,
  refId: string,
  range: TimeRange
): Promise<TrendPoint[]> {
  const request = {
    targets: [{ refId, expr, instant: false, range: true, format: 'time_series' }],
    range,
    intervalMs: 15000,
    maxDataPoints: 50,
    requestId: `svcmap-trend-${refId}`,
  } as unknown as DataQueryRequest;

  const frames = await new Promise<DataFrame[]>((resolve, reject) => {
    (ds as any).query(request).subscribe({
      next: (response: { data: DataFrame[] }) => resolve(response.data ?? []),
      error: (err: unknown) => reject(err),
    });
  });

  const points: TrendPoint[] = [];
  for (const frame of frames) {
    const timeField = frame.fields.find((f) => f.type === FieldType.time);
    const valueField = frame.fields.find((f) => f.type === FieldType.number);
    if (!timeField || !valueField) {
      continue;
    }
    for (let i = 0; i < frame.length; i++) {
      const t = Number(timeField.values[i]);
      const v = Number(valueField.values[i]);
      if (!isNaN(t) && !isNaN(v)) {
        points.push({ t, v });
      }
    }
  }
  // Combine all frames' points and sort by time
  points.sort((a, b) => a.t - b.t);
  return points;
}

export function useEdgeTrends(
  selectedEdge: SelectedEdge | null,
  promDatasource: string,
  timeRange: TimeRange
): { rpsSeries: TrendPoint[]; latSeries: TrendPoint[] } {
  const [rpsSeries, setRpsSeries] = useState<TrendPoint[]>([]);
  const [latSeries, setLatSeries] = useState<TrendPoint[]>([]);

  useEffect(() => {
    if (!selectedEdge) {
      setRpsSeries([]);
      setLatSeries([]);
      return;
    }

    let cancelled = false;
    const src = escapeProm(selectedEdge.sourceAppId || selectedEdge.source);
    const tgt = escapeProm(selectedEdge.targetAppId || selectedEdge.target);

    async function fetch() {
      try {
        const ds = await getDataSourceSrv().get(promDatasource || undefined);
        const rpsExpr = `sum(rr_connection_l7_requests{app_id="${src}", destination="${tgt}"})`;
        const latExpr = `sum(rr_connection_l7_latency{app_id="${src}", destination="${tgt}"})`;

        const [rps, lat] = await Promise.all([
          runRangeQuery(ds, rpsExpr, 'trend-rps', timeRange),
          runRangeQuery(ds, latExpr, 'trend-lat', timeRange),
        ]);
        if (!cancelled) {
          setRpsSeries(rps);
          setLatSeries(lat);
        }
      } catch {
        // Trend sparklines are best-effort
      }
    }

    fetch();
    return () => { cancelled = true; };
  }, [selectedEdge?.source, selectedEdge?.target, promDatasource, timeRange]);

  return { rpsSeries, latSeries };
}
