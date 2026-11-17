import { css, cx } from '@emotion/css';
import { config } from '@grafana/runtime';
import { TimeRange } from '@grafana/data';
import { TraceRow, TraceFilter, SelectedEdge } from '../../types';
import { TraceFilterChips } from './TraceFilters';

interface Props {
  traces: TraceRow[];
  loading: boolean;
  error: string | null;
  selectedEdge: SelectedEdge | null;
  selectedNodeLabel?: string | null;
  filter: TraceFilter;
  onFilterChange: (f: TraceFilter) => void;
  timeRange: TimeRange;
  tracesDashboardUid: string;
  tracesViewPanel: number;
}

const s = {
  container: css`
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
    overflow: hidden;
    background: #0e0e1c;
    border-top: 1px solid #2a2a3a;
  `,
  header: css`
    padding: 8px 14px 0;
    display: flex;
    align-items: center;
    justify-content: space-between;
    flex-shrink: 0;
  `,
  title: css`
    font-size: 13px;
    font-weight: 600;
    color: #bbb;
  `,
  tableWrap: css`
    flex: 1;
    min-height: 0;
    overflow-x: auto;
    overflow-y: auto;
    -webkit-overflow-scrolling: touch;
    padding: 0 14px 14px;
  `,
  table: css`
    width: 100%;
    border-collapse: collapse;
    font-size: 11px;
  `,
  th: css`
    text-align: left;
    padding: 6px 8px;
    color: #777;
    font-weight: 600;
    border-bottom: 1px solid #2a2a3a;
    white-space: nowrap;
    position: sticky;
    top: 0;
    background: #0e0e1c;
    font-size: 10px;
    text-transform: uppercase;
    letter-spacing: 0.3px;
  `,
  td: css`
    padding: 5px 8px;
    color: #ccc;
    border-bottom: 1px solid rgba(255, 255, 255, 0.03);
    white-space: nowrap;
    max-width: 200px;
    overflow: hidden;
    text-overflow: ellipsis;
  `,
  traceLink: css`
    color: #73a5f0;
    text-decoration: none;
    font-family: 'JetBrains Mono', monospace;
    font-size: 10px;
    &:hover {
      text-decoration: underline;
      color: #99bbff;
    }
  `,
  statusOk: css`
    color: #73bf69;
  `,
  statusError: css`
    color: #f2495c;
    font-weight: 600;
  `,
  placeholder: css`
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: #555;
    font-size: 13px;
  `,
  durationBar: (pct: number, isErr: boolean) => css`
    display: inline-block;
    height: 4px;
    width: ${Math.max(4, pct)}%;
    max-width: 80px;
    background: ${isErr ? '#f2495c' : '#73bf69'};
    border-radius: 2px;
    margin-right: 6px;
    vertical-align: middle;
  `,
};

function isError(status: string): boolean {
  // OTLP StatusCode: 'Error', 'STATUS_CODE_ERROR', or numeric '2' (= ERROR in proto enum)
  return status === 'Error' || status === 'STATUS_CODE_ERROR' || status === '2';
}

function buildTraceUrl(
  traceId: string,
  timeRange: TimeRange,
  dashboardUid: string,
  viewPanel: number
): string {
  const q = new URLSearchParams();
  q.set('from', String(timeRange.from.valueOf()));
  q.set('to', String(timeRange.to.valueOf()));
  q.set('timezone', 'browser');
  q.set('var-trace_id', traceId);
  q.set('viewPanel', String(viewPanel));
  return `${config.appSubUrl}/d/${encodeURIComponent(dashboardUid)}?${q.toString()}`;
}

export function TraceTable({
  traces,
  loading,
  error,
  selectedEdge,
  selectedNodeLabel,
  filter,
  onFilterChange,
  timeRange,
  tracesDashboardUid,
  tracesViewPanel,
}: Props) {
  const hasSelection = !!selectedEdge || !!selectedNodeLabel;
  if (!hasSelection) {
    return (
      <div className={s.container} style={{ minHeight: 100 }}>
        <div className={s.placeholder}>Click an edge or node to explore traces</div>
      </div>
    );
  }

  const traceTitle = selectedEdge
    ? `Traces: ${selectedEdge.sourceLabel} → ${selectedEdge.targetLabel}`
    : `Traces: ${selectedNodeLabel}`;
  const maxDuration = traces.reduce((m, t) => Math.max(m, t.durationMs), 1);

  return (
    <div className={s.container}>
      <div className={s.header}>
        <div className={s.title}>{traceTitle}</div>
        <TraceFilterChips active={filter} onChange={onFilterChange} />
      </div>
      <div className={s.tableWrap}>
        {loading && <div className={s.placeholder}>Loading traces...</div>}
        {error && <div className={s.placeholder} style={{ color: '#f2495c' }}>{error}</div>}
        {!loading && !error && traces.length === 0 && (
          <div className={s.placeholder}>No traces found</div>
        )}
        {!loading && !error && traces.length > 0 && (
          <table className={s.table}>
            <thead>
              <tr>
                <th className={s.th}>Time</th>
                <th className={s.th}>Trace ID</th>
                <th className={s.th}>Service</th>
                <th className={s.th}>Span</th>
                <th className={s.th}>Status</th>
                <th className={s.th}>Duration</th>
              </tr>
            </thead>
            <tbody>
              {traces.map((t, i) => {
                const err = isError(t.statusCode);
                const pct = (t.durationMs / maxDuration) * 100;
                let tsMs = Number(t.timestamp);
                if (tsMs > 1e15) {
                  tsMs = tsMs / 1e6;
                } else if (tsMs < 1e12 && tsMs > 1e9) {
                  tsMs = tsMs * 1000;
                }
                const ts = isNaN(tsMs) ? new Date(t.timestamp) : new Date(tsMs);
                const timeStr = isNaN(ts.getTime())
                  ? t.timestamp
                  : ts.toLocaleString(undefined, {
                      month: 'short', day: 'numeric',
                      hour: '2-digit', minute: '2-digit', second: '2-digit',
                      hour12: false,
                    });

                return (
                  <tr key={`${t.traceId}-${i}`}>
                    <td className={s.td}>{timeStr}</td>
                    <td className={s.td}>
                      <a
                        className={s.traceLink}
                        href={buildTraceUrl(t.traceId, timeRange, tracesDashboardUid, tracesViewPanel)}
                        target="_blank"
                        rel="noopener noreferrer"
                      >
                        {t.traceId.substring(0, 16)}…
                      </a>
                    </td>
                    <td className={s.td}>{t.serviceName}</td>
                    <td className={s.td}>{t.spanName}</td>
                    <td className={cx(s.td, err ? s.statusError : s.statusOk)}>
                      {err ? 'ERROR' : 'OK'}
                    </td>
                    <td className={s.td}>
                      <span className={s.durationBar(pct, err)} />
                      {t.durationMs.toFixed(1)} ms
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
