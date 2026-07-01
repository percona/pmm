import { css } from '@emotion/css';
import { HEALTH_COLORS, SLOW_THRESHOLD_MS } from '../../constants';
import { SelectedEdge, ServiceMapOptions } from '../../types';

interface Props {
  edge: SelectedEdge;
  options: ServiceMapOptions;
  onClose: () => void;
  rpsSeries?: Array<{ t: number; v: number }>;
  latSeries?: Array<{ t: number; v: number }>;
}

function formatBytes(b: number): string {
  if (b > 1e9) {
    return `${(b / 1e9).toFixed(2)} GB/s`;
  }
  if (b > 1e6) {
    return `${(b / 1e6).toFixed(2)} MB/s`;
  }
  if (b > 1e3) {
    return `${(b / 1e3).toFixed(1)} KB/s`;
  }
  return `${b.toFixed(0)} B/s`;
}

function Sparkline({ points, color, height = 32, width = 220 }: {
  points: Array<{ t: number; v: number }>;
  color: string;
  height?: number;
  width?: number;
}) {
  if (points.length < 2) {
    return null;
  }
  const maxV = Math.max(...points.map((p) => p.v), 0.001);
  const minT = points[0].t;
  const maxT = points[points.length - 1].t;
  const rangeT = maxT - minT || 1;

  const pathParts = points.map((p, i) => {
    const x = ((p.t - minT) / rangeT) * width;
    const y = height - (p.v / maxV) * (height - 4) - 2;
    return `${i === 0 ? 'M' : 'L'}${x.toFixed(1)},${y.toFixed(1)}`;
  });

  return (
    <svg width={width} height={height} style={{ display: 'block', margin: '4px 0' }}>
      <path d={pathParts.join(' ')} fill="none" stroke={color} strokeWidth={1.5} opacity={0.8} />
    </svg>
  );
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
  flow: css`
    font-size: 12px;
    color: #aaa;
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 8px 0;
    border-bottom: 1px solid #2a2a4a;
  `,
  arrow: css`
    color: #555;
    font-size: 14px;
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
  whySection: css`
    margin-top: 8px;
    padding: 10px;
    border-radius: 6px;
    font-size: 11px;
    line-height: 1.5;
  `,
  healthDot: (color: string) => css`
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: ${color};
    box-shadow: 0 0 4px ${color}88;
  `,
};

export function EdgeDetailSidebar({ edge, options, onClose, rpsSeries, latSeries }: Props) {
  const { edge: e } = edge;
  const color = HEALTH_COLORS[e.health];
  const reasons: string[] = [];

  if (e.errPct >= options.errorRedThreshold) {
    reasons.push(`Error rate ${e.errPct.toFixed(1)}% exceeds ${options.errorRedThreshold}% red threshold`);
  } else if (e.errPct >= options.errorAmberThreshold) {
    reasons.push(`Error rate ${e.errPct.toFixed(1)}% exceeds ${options.errorAmberThreshold}% amber threshold`);
  }
  if (e.p95Ms > SLOW_THRESHOLD_MS) {
    reasons.push(`p95 latency ${e.p95Ms.toFixed(0)} ms exceeds ${SLOW_THRESHOLD_MS} ms`);
  }

  return (
    <div className={s.sidebar}>
      <div className={s.header}>
        <div className={s.title}>
          <span className={s.healthDot(color)} />
          Edge Detail
        </div>
        <button className={s.closeBtn} onClick={onClose} aria-label="Close">×</button>
      </div>

      <div className={s.flow}>
        <span>{edge.sourceLabel}</span>
        <span className={s.arrow}>→</span>
        <span>{edge.targetLabel}</span>
      </div>

      <div className={s.section}>
        <div className={s.sectionLabel}>Metrics</div>
        <div className={s.metricRow}>
          <span className={s.metricLabel}>Requests/s</span>
          <span className={s.metricValue}>{e.rps.toFixed(2)}</span>
        </div>
        {rpsSeries && rpsSeries.length > 1 && (
          <Sparkline points={rpsSeries} color="#73a5f0" />
        )}
        <div className={s.metricRow}>
          <span className={s.metricLabel}>p95 Latency</span>
          <span className={s.metricValue}>{e.p95Ms.toFixed(1)} ms</span>
        </div>
        {latSeries && latSeries.length > 1 && (
          <Sparkline points={latSeries} color="#ff9830" />
        )}
        <div className={s.metricRow}>
          <span className={s.metricLabel}>Error %</span>
          <span className={s.metricValue} style={{ color }}>
            {e.errPct.toFixed(2)}%
          </span>
        </div>
        <div className={s.metricRow}>
          <span className={s.metricLabel}>Bytes Out</span>
          <span className={s.metricValue}>{formatBytes(e.bytesOut)}</span>
        </div>
        <div className={s.metricRow}>
          <span className={s.metricLabel}>Bytes In</span>
          <span className={s.metricValue}>{formatBytes(e.bytesIn)}</span>
        </div>
      </div>

      {reasons.length > 0 && (
        <div
          className={s.whySection}
          style={{ background: `${color}15`, border: `1px solid ${color}30` }}
        >
          <strong style={{ color }}>Why {e.health}?</strong>
          <ul style={{ margin: '4px 0 0', paddingLeft: 16 }}>
            {reasons.map((r, i) => (
              <li key={i}>{r}</li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}
