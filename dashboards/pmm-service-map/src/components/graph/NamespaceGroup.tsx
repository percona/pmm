import { memo } from 'react';
import { type NodeProps } from '@xyflow/react';
import { css } from '@emotion/css';

interface NamespaceGroupData {
  label: string;
  [key: string]: unknown;
}

const styles = {
  container: css`
    background: rgba(30, 30, 50, 0.3);
    border: 1px solid rgba(100, 100, 140, 0.25);
    border-radius: 12px;
    width: 100%;
    height: 100%;
    position: relative;
    pointer-events: none;
  `,
  label: css`
    position: absolute;
    top: 10px;
    left: 14px;
    font-size: 10px;
    font-weight: 700;
    color: rgba(160, 160, 200, 0.7);
    text-transform: uppercase;
    letter-spacing: 1px;
  `,
};

export const NamespaceGroup = memo(function NamespaceGroup({ data }: NodeProps) {
  const { label } = data as unknown as NamespaceGroupData;

  return (
    <div className={styles.container}>
      <div className={styles.label}>{label}</div>
    </div>
  );
});
