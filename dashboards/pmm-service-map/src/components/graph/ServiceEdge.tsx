import { memo, useState, type CSSProperties } from 'react';
import {
  EdgeLabelRenderer,
  getBezierPath,
  type EdgeProps,
} from '@xyflow/react';
import { css } from '@emotion/css';
import { HEALTH_COLORS, EDGE_MIN_WIDTH } from '../../constants';
import { HealthStatus } from '../../types';

interface ServiceEdgeData {
  rps: number;
  errPct: number;
  p95Ms: number;
  bytesIn: number;
  bytesOut: number;
  health: HealthStatus;
  sourceLabel?: string;
  targetLabel?: string;
  [key: string]: unknown;
}

function formatBytes(b: number): string {
  if (b > 1e9) {
    return `${(b / 1e9).toFixed(1)} GB/s`;
  }
  if (b > 1e6) {
    return `${(b / 1e6).toFixed(1)} MB/s`;
  }
  if (b > 1e3) {
    return `${(b / 1e3).toFixed(1)} KB/s`;
  }
  return `${b.toFixed(0)} B/s`;
}

const tooltipStyle = css`
  background: #1e1e30;
  border: 1px solid #4a4a6a;
  border-radius: 8px;
  padding: 10px 14px;
  color: #e0e0e0;
  font-size: 11px;
  line-height: 1.6;
  pointer-events: none;
  white-space: nowrap;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.5);
  z-index: 100;
`;

export const ServiceEdge = memo(function ServiceEdge({
  id,
  sourceX,
  sourceY,
  targetX,
  targetY,
  sourcePosition,
  targetPosition,
  style = {},
  data,
}: EdgeProps) {
  const [hovered, setHovered] = useState(false);
  const edgeData = data as unknown as ServiceEdgeData;

  const [edgePath, labelX, labelY] = getBezierPath({
    sourceX,
    sourceY,
    sourcePosition,
    targetX,
    targetY,
    targetPosition,
  });

  const health = edgeData?.health ?? 'unknown';
  const color = HEALTH_COLORS[health];
  const edgeStyle = style as CSSProperties;
  const strokeWidth = (edgeStyle?.strokeWidth as number) ?? EDGE_MIN_WIDTH;
  /** React Flow passes dimming here when a node is selected; must apply to the whole edge (incl. marker). */
  const dim = typeof edgeStyle.opacity === 'number' ? edgeStyle.opacity : 1;

  return (
    <>
      <g style={{ opacity: dim }}>
        {/* Wide transparent hit area for hover/click */}
        <path
          d={edgePath}
          fill="none"
          stroke="transparent"
          strokeWidth={Math.max(20, strokeWidth + 12)}
          onMouseEnter={() => setHovered(true)}
          onMouseLeave={() => setHovered(false)}
          style={{ cursor: 'pointer' }}
        />
        {/* Visible edge path — uses health color and thickness from layout */}
        <path
          d={edgePath}
          fill="none"
          stroke={hovered ? '#fff' : color}
          strokeWidth={strokeWidth}
          strokeOpacity={hovered ? 1 : 0.7}
          markerEnd={`url(#arrow-${health})`}
          style={{ pointerEvents: 'none', transition: 'stroke 0.15s, stroke-opacity 0.15s' }}
        />
      </g>
      {hovered && edgeData && (
        <EdgeLabelRenderer>
          <div
            className={tooltipStyle}
            style={{
              position: 'absolute',
              transform: `translate(-50%, -100%) translate(${labelX}px, ${labelY - 12}px)`,
              opacity: Math.max(dim, 0.35),
            }}
          >
            {edgeData.sourceLabel && edgeData.targetLabel && (
              <div style={{ fontWeight: 600, marginBottom: 4, color: '#fff' }}>
                {edgeData.sourceLabel} → {edgeData.targetLabel}
              </div>
            )}
            <div><span style={{ color: '#888' }}>RPS:</span> <strong>{edgeData.rps.toFixed(2)}</strong></div>
            <div><span style={{ color: '#888' }}>p95:</span> {edgeData.p95Ms.toFixed(1)} ms</div>
            <div>
              <span style={{ color: '#888' }}>Errors:</span>{' '}
              <span style={{ color: edgeData.errPct > 0 ? HEALTH_COLORS.red : '#73bf69' }}>
                {edgeData.errPct.toFixed(2)}%
              </span>
            </div>
            {edgeData.bytesOut > 0 && <div><span style={{ color: '#888' }}>Out:</span> {formatBytes(edgeData.bytesOut)}</div>}
            {edgeData.bytesIn > 0 && <div><span style={{ color: '#888' }}>In:</span> {formatBytes(edgeData.bytesIn)}</div>}
          </div>
        </EdgeLabelRenderer>
      )}
    </>
  );
});
