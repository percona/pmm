import { css } from '@emotion/css';
import { HEALTH_COLORS } from '../../constants';
import { SelectedNode } from '../../types';
import { formatNodeLabel } from '../../data/parseAppId';

interface Props {
  node: SelectedNode;
  onClose: () => void;
}

function formatRps(rps: number): string {
  if (rps >= 1000) {
    return `${(rps / 1000).toFixed(1)}k`;
  }
  return rps.toFixed(2);
}

const s = {
  sidebar: css`
    width: 280px;
    background: #151525;
    border-left: 1px solid #2a2a4a;
    padding: 16px;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 12px;
    flex-shrink: 0;
  `,
  header: css`
    display: flex;
    justify-content: space-between;
    align-items: center;
  `,
  title: css`
    font-size: 14px;
    font-weight: 600;
    color: #e0e0e0;
    display: flex;
    align-items: center;
    gap: 6px;
  `,
  closeBtn: css`
    background: none;
    border: none;
    color: #666;
    cursor: pointer;
    font-size: 18px;
    line-height: 1;
    padding: 4px;
    &:hover { color: #fff; }
  `,
  section: css`
    margin-top: 4px;
  `,
  sectionLabel: css`
    font-size: 10px;
    font-weight: 600;
    color: #666;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    margin-bottom: 6px;
  `,
  metricRow: css`
    display: flex;
    justify-content: space-between;
    font-size: 12px;
    color: #ccc;
    padding: 4px 0;
  `,
  metricLabel: css`
    color: #777;
  `,
  metricValue: css`
    font-weight: 600;
    font-variant-numeric: tabular-nums;
  `,
  edgeItem: css`
    padding: 8px 10px;
    background: #1a1a2e;
    border: 1px solid #2a2a4a;
    border-radius: 6px;
    margin-bottom: 6px;
  `,
  edgeTarget: css`
    font-size: 11px;
    font-weight: 600;
    color: #ddd;
    margin-bottom: 4px;
    display: flex;
    align-items: center;
    gap: 6px;
  `,
  edgeMetrics: css`
    display: flex;
    gap: 12px;
    font-size: 10px;
    color: #999;
  `,
  healthDot: (color: string) => css`
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: ${color};
    box-shadow: 0 0 4px ${color}88;
    flex-shrink: 0;
  `,
};

export function NodeDetailSidebar({ node, onClose }: Props) {
  const { node: n, outgoingEdges, outgoingLabels, childContainers, internalSamePodEdgesHidden } = node;
  const color = HEALTH_COLORS[n.health];

  const totalRps = outgoingEdges.reduce((sum, e) => sum + e.rps, 0);
  const totalErrRps = outgoingEdges.reduce((sum, e) => sum + e.rps * e.errPct / 100, 0);
  const avgErrPct = totalRps > 0 ? (totalErrRps / totalRps) * 100 : 0;

  return (
    <div className={s.sidebar}>
      <div className={s.header}>
        <div className={s.title}>
          <span className={s.healthDot(color)} />
          {node.label}
        </div>
        <button className={s.closeBtn} onClick={onClose} aria-label="Close">×</button>
      </div>

      <div className={s.section}>
        <div className={s.sectionLabel}>Node Metrics</div>
        <div className={s.metricRow}>
          <span className={s.metricLabel}>Requests/s</span>
          <span className={s.metricValue}>{formatRps(n.rps)}</span>
        </div>
        <div className={s.metricRow}>
          <span className={s.metricLabel}>Error %</span>
          <span className={s.metricValue} style={{ color }}>{n.errPct.toFixed(2)}%</span>
        </div>
        <div className={s.metricRow}>
          <span className={s.metricLabel}>p95 Latency</span>
          <span className={s.metricValue}>{n.p95Ms.toFixed(1)} ms</span>
        </div>
      </div>

      {childContainers && childContainers.length > 0 && (
        <div className={s.section}>
          <div className={s.sectionLabel}>Containers ({childContainers.length})</div>
          {childContainers.map((c) => {
            const cColor = HEALTH_COLORS[c.health];
            return (
              <div key={c.id} className={s.edgeItem}>
                <div className={s.edgeTarget}>
                  <span className={s.healthDot(cColor)} />
                  {formatNodeLabel(c.parsed, 'name')}
                </div>
                <div className={s.edgeMetrics}>
                  <span>{formatRps(c.rps)} req/s</span>
                  <span style={{ color: c.errPct > 0 ? HEALTH_COLORS.red : '#999' }}>
                    {c.errPct.toFixed(1)}% err
                  </span>
                  <span>{c.p95Ms.toFixed(1)} ms</span>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {internalSamePodEdgesHidden != null && internalSamePodEdgesHidden > 0 && (
        <div style={{ fontSize: 10, color: '#777', lineHeight: 1.4 }}>
          Pod view hides {internalSamePodEdgesHidden} same-pod container↔container edge
          {internalSamePodEdgesHidden === 1 ? '' : 's'} (turn off Group by pod to see them).
        </div>
      )}

      <div className={s.section}>
        <div className={s.sectionLabel}>
          Outgoing Edges ({outgoingEdges.length})
          {totalRps > 0 && <span style={{ color: '#999', fontWeight: 400 }}> — {formatRps(totalRps)} req/s total, {avgErrPct.toFixed(1)}% err</span>}
        </div>
        {outgoingEdges.map((e, i) => {
          const eColor = HEALTH_COLORS[e.health];
          return (
            <div key={e.id} className={s.edgeItem}>
              <div className={s.edgeTarget}>
                <span className={s.healthDot(eColor)} />
                → {outgoingLabels[i]}
              </div>
              <div className={s.edgeMetrics}>
                <span>{formatRps(e.rps)} req/s</span>
                <span style={{ color: e.errPct > 0 ? HEALTH_COLORS.red : '#999' }}>
                  {e.errPct.toFixed(1)}% err
                </span>
                <span>{e.p95Ms.toFixed(1)} ms</span>
              </div>
            </div>
          );
        })}
        {outgoingEdges.length === 0 && (
          <div style={{ fontSize: 11, color: '#666', padding: '8px 0' }}>No outgoing edges</div>
        )}
      </div>
    </div>
  );
}
