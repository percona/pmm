import { memo, useState, useCallback } from 'react';
import { createPortal } from 'react-dom';
import { Handle, Position, type NodeProps } from '@xyflow/react';
import { css } from '@emotion/css';
import { HEALTH_COLORS, NODE_WIDTH, NODE_HEIGHT } from '../../constants';
import { HealthStatus } from '../../types';

interface ServiceNodeData {
  label: string;
  rps: number;
  errPct: number;
  p95Ms: number;
  health: HealthStatus;
  fullId: string;
  namespace: string;
  bytesIn: number;
  bytesOut: number;
  groupByPod?: boolean;
  podChildContainerCount?: number;
  [key: string]: unknown;
}

function formatRps(rps: number): string {
  if (rps >= 1000) {
    return `${(rps / 1000).toFixed(1)}k`;
  }
  return rps.toFixed(2);
}

function formatBytes(b: number): string {
  if (b > 1e6) {
    return `${(b / 1e6).toFixed(1)} MB/s`;
  }
  if (b > 1e3) {
    return `${(b / 1e3).toFixed(1)} KB/s`;
  }
  return `${b.toFixed(0)} B/s`;
}

const styles = {
  container: (borderColor: string) => css`
    width: ${NODE_WIDTH}px;
    height: ${NODE_HEIGHT}px;
    background: linear-gradient(135deg, #1a1a2e 0%, #1e1e35 100%);
    border: 1.5px solid ${borderColor};
    border-radius: 8px;
    display: flex;
    align-items: center;
    padding: 0 12px;
    gap: 8px;
    cursor: pointer;
    transition: border-color 0.15s, box-shadow 0.15s;
    &:hover {
      border-color: #fff;
      box-shadow: 0 0 12px rgba(255, 255, 255, 0.15);
    }
  `,
  dot: (color: string) => css`
    width: 10px;
    height: 10px;
    border-radius: 50%;
    background: ${color};
    flex-shrink: 0;
    box-shadow: 0 0 6px ${color}66;
  `,
  content: css`
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 1px;
  `,
  name: css`
    font-size: 11px;
    font-weight: 600;
    color: #e8e8f0;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  `,
  metric: css`
    font-size: 9px;
    color: #9999bb;
    display: flex;
    gap: 8px;
  `,
  tooltip: css`
    position: fixed;
    background: #1e1e30;
    border: 1px solid #4a4a6a;
    border-radius: 10px;
    padding: 12px 16px;
    color: #e0e0e0;
    font-size: 11px;
    line-height: 1.5;
    white-space: nowrap;
    box-shadow: 0 6px 24px rgba(0, 0, 0, 0.6);
    z-index: 10000;
    pointer-events: none;
  `,
  redMetrics: css`
    display: flex;
    gap: 16px;
    margin-top: 8px;
    padding-top: 8px;
    border-top: 1px solid #3a3a5a;
  `,
  redBox: css`
    text-align: center;
  `,
  redLabel: css`
    font-size: 9px;
    font-weight: 700;
    color: #666;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  `,
  redValue: css`
    font-size: 14px;
    font-weight: 700;
    margin-top: 2px;
  `,
  redUnit: css`
    font-size: 9px;
    color: #888;
    font-weight: 400;
  `,
  handle: css`
    visibility: hidden;
    width: 1px;
    height: 1px;
  `,
};

export const ServiceNode = memo(function ServiceNode({ data }: NodeProps) {
  const {
    label, rps, errPct, p95Ms, health, fullId, namespace, bytesIn, bytesOut,
    groupByPod, podChildContainerCount,
  } = data as unknown as ServiceNodeData;
  const color = HEALTH_COLORS[health] || HEALTH_COLORS.unknown;
  const [hovered, setHovered] = useState(false);
  const [tooltipPos, setTooltipPos] = useState({ x: 0, y: 0 });

  const handleMouseEnter = useCallback((e: React.MouseEvent) => {
    setHovered(true);
    setTooltipPos({ x: e.clientX + 12, y: e.clientY - 10 });
  }, []);

  const handleMouseMove = useCallback((e: React.MouseEvent) => {
    setTooltipPos({ x: e.clientX + 12, y: e.clientY - 10 });
  }, []);

  const handleMouseLeave = useCallback(() => setHovered(false), []);

  return (
    <div
      className={styles.container(color + '66')}
      onMouseEnter={handleMouseEnter}
      onMouseMove={handleMouseMove}
      onMouseLeave={handleMouseLeave}
    >
      <Handle type="target" position={Position.Left} className={styles.handle} />
      <div className={styles.dot(color)} />
      <div className={styles.content}>
        <div className={styles.name}>{label}</div>
        <div className={styles.metric}>
          <span>{formatRps(rps)} req/s</span>
          {errPct > 0 && <span style={{ color: HEALTH_COLORS.red }}>{errPct.toFixed(1)}% err</span>}
        </div>
      </div>
      <Handle type="source" position={Position.Right} className={styles.handle} />
      {hovered && createPortal(
        <div className={styles.tooltip} style={{ left: tooltipPos.x, top: tooltipPos.y }}>
          <div style={{ fontWeight: 700, color: '#fff', marginBottom: 2, fontSize: 12 }}>{label}</div>
          {namespace && namespace !== 'external' && (
            <div style={{ color: '#888', fontSize: 10, marginBottom: 4 }}>{fullId}</div>
          )}
          {groupByPod && (podChildContainerCount ?? 0) > 1 && (
            <div style={{ color: '#a0a0c8', fontSize: 10, marginBottom: 4 }}>
              {podChildContainerCount} containers — click for detail
            </div>
          )}
          {(bytesIn > 0 || bytesOut > 0) && (
            <div style={{ color: '#888', fontSize: 10 }}>
              {bytesIn > 0 && <span>In: {formatBytes(bytesIn)} </span>}
              {bytesOut > 0 && <span>Out: {formatBytes(bytesOut)}</span>}
            </div>
          )}
          <div className={styles.redMetrics}>
            <div className={styles.redBox}>
              <div className={styles.redLabel}>Rate</div>
              <div className={styles.redValue}>
                {formatRps(rps)}<span className={styles.redUnit}> req/s</span>
              </div>
            </div>
            <div className={styles.redBox}>
              <div className={styles.redLabel}>Errors</div>
              <div className={styles.redValue} style={{ color: errPct > 0 ? HEALTH_COLORS.red : HEALTH_COLORS.green }}>
                {errPct.toFixed(2)}<span className={styles.redUnit}>%</span>
              </div>
            </div>
            <div className={styles.redBox}>
              <div className={styles.redLabel}>Duration</div>
              <div className={styles.redValue}>
                {p95Ms.toFixed(1)}<span className={styles.redUnit}> ms</span>
              </div>
            </div>
          </div>
        </div>,
        document.body
      )}
    </div>
  );
});
