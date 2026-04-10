import { useEffect, useState } from 'react';
import { getDataSourceSrv } from '@grafana/runtime';
import { DataFrame, DataQueryRequest, TimeRange } from '@grafana/data';
import { SelectedEdge, SelectedNode, TraceFilter, TraceRow } from '../types';
import { SLOW_THRESHOLD_MS, TRACE_LIMIT } from '../constants';
import {
  buildNodeTraceServiceWhereClause,
  collectNodeTraceNameIds,
  expandOtelServiceNameCandidates,
  expandTraceSearchTokens,
} from './traceSql';

function escapeSql(s: string): string {
  return s.replace(/'/g, "''");
}

function buildEdgeSQL(edge: SelectedEdge, filter: TraceFilter): string {
  const expanded = expandOtelServiceNameCandidates([
    edge.sourceAppId || edge.source,
    edge.targetAppId || edge.target,
  ]);
  const uniq = [...new Set(expanded.filter(Boolean))].slice(0, 200);
  let where =
    uniq.length > 0
      ? `ServiceName IN (${uniq.map((id) => `'${escapeSql(id)}'`).join(', ')})`
      : '1 = 0';

  if (filter === 'errors') {
    where = `${where} AND StatusCode IN ('Error', 'STATUS_CODE_ERROR', '2')`;
  } else if (filter === 'slow') {
    where = `${where} AND Duration > ${SLOW_THRESHOLD_MS * 1_000_000}`;
  }

  return `
SELECT
  Timestamp,
  TraceId,
  ServiceName,
  SpanName,
  StatusCode,
  Duration / 1000000 AS duration_ms
FROM otel.otel_traces
WHERE $__timeFilter(Timestamp)
  AND (${where})
ORDER BY Timestamp DESC
LIMIT ${TRACE_LIMIT}
  `.trim();
}

function buildNodeSQL(node: SelectedNode, filter: TraceFilter): string {
  const ids = collectNodeTraceNameIds(node);
  const expanded = expandOtelServiceNameCandidates(ids);
  const tokens = expandTraceSearchTokens(expanded);
  let where = buildNodeTraceServiceWhereClause(expanded, tokens);

  if (filter === 'errors') {
    where = `(${where}) AND StatusCode IN ('Error', 'STATUS_CODE_ERROR', '2')`;
  } else if (filter === 'slow') {
    where = `(${where}) AND Duration > ${SLOW_THRESHOLD_MS * 1_000_000}`;
  }

  return `
SELECT
  Timestamp,
  TraceId,
  ServiceName,
  SpanName,
  StatusCode,
  Duration / 1000000 AS duration_ms
FROM otel.otel_traces
WHERE $__timeFilter(Timestamp)
  AND ${where}
ORDER BY Timestamp DESC
LIMIT ${TRACE_LIMIT}
  `.trim();
}

function frameToTraceRows(frames: DataFrame[]): TraceRow[] {
  if (!frames.length) {
    return [];
  }
  const frame = frames[0];
  const getCol = (name: string) =>
    frame.fields.find((f) => f.name.toLowerCase() === name.toLowerCase());

  const ts = getCol('Timestamp') ?? getCol('timestamp');
  const tid = getCol('TraceId') ?? getCol('traceid');
  const svc = getCol('ServiceName') ?? getCol('servicename');
  const span = getCol('SpanName') ?? getCol('spanname');
  const status = getCol('StatusCode') ?? getCol('statuscode');
  const dur = getCol('duration_ms');

  if (!ts || !tid) {
    return [];
  }

  const rows: TraceRow[] = [];
  for (let i = 0; i < frame.length; i++) {
    rows.push({
      timestamp: String(ts.values[i]),
      traceId: String(tid.values[i]),
      serviceName: svc ? String(svc.values[i]) : '',
      spanName: span ? String(span.values[i]) : '',
      statusCode: status ? String(status.values[i]) : '',
      durationMs: dur ? Number(dur.values[i]) : 0,
    });
  }
  return rows;
}

export function useTraceData(
  selectedEdge: SelectedEdge | null,
  selectedNode: SelectedNode | null,
  filter: TraceFilter,
  clickhouseDatasource: string,
  timeRange: TimeRange
): { traces: TraceRow[]; loading: boolean; error: string | null } {
  const [traces, setTraces] = useState<TraceRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const hasSelection = !!selectedEdge || !!selectedNode;
  const nodeTraceKey =
    !selectedNode
      ? ''
      : `${selectedNode.id}\x1f${(selectedNode.traceServiceNames ?? []).join('\x1f')}`;

  useEffect(() => {
    if (!hasSelection) {
      setTraces([]);
      setError(null);
      return;
    }

    let cancelled = false;

    async function fetchTraces() {
      setLoading(true);
      setError(null);

      try {
        const dsSrv = getDataSourceSrv();
        const ds = await dsSrv.get(clickhouseDatasource || undefined);

        const sql = selectedEdge
          ? buildEdgeSQL(selectedEdge, filter)
          : buildNodeSQL(selectedNode!, filter);

        const request = {
          targets: [{ refId: 'traces', rawSql: sql, format: 1 }],
          range: timeRange,
          intervalMs: 1000,
          maxDataPoints: TRACE_LIMIT,
          requestId: 'svcmap-traces',
        } as unknown as DataQueryRequest;

        const frames = await new Promise<DataFrame[]>((resolve, reject) => {
          (ds as any).query(request).subscribe({
            next: (response: { data: DataFrame[] }) => resolve(response.data ?? []),
            error: (err: unknown) => reject(err),
          });
        });

        if (!cancelled) {
          setTraces(frameToTraceRows(frames));
        }
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

    fetchTraces();
    return () => {
      cancelled = true;
    };
  }, [
    selectedEdge?.source,
    selectedEdge?.target,
    nodeTraceKey,
    filter,
    clickhouseDatasource,
    timeRange,
    hasSelection,
  ]);

  return { traces, loading, error };
}
