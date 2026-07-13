import { css, cx } from '@emotion/css';
import { TraceFilter } from '../../types';
import { SLOW_THRESHOLD_MS } from '../../constants';

interface Props {
  active: TraceFilter;
  onChange: (f: TraceFilter) => void;
}

const FILTERS: Array<{ value: TraceFilter; label: string }> = [
  { value: 'all', label: 'All' },
  { value: 'errors', label: 'Errors' },
  { value: 'slow', label: `Slow >${SLOW_THRESHOLD_MS}ms` },
];

const styles = {
  container: css`
    display: flex;
    gap: 6px;
    padding: 8px 0;
  `,
  chip: css`
    padding: 4px 12px;
    border-radius: 16px;
    font-size: 11px;
    font-weight: 500;
    cursor: pointer;
    border: 1px solid #3a3a5a;
    background: transparent;
    color: #aaa;
    transition: all 0.15s;
    &:hover {
      border-color: #6e6e8e;
      color: #e0e0e0;
    }
  `,
  active: css`
    background: #3a3a5a;
    color: #e0e0e0;
    border-color: #6e6e8e;
  `,
};

export function TraceFilterChips({ active, onChange }: Props) {
  return (
    <div className={styles.container}>
      {FILTERS.map((f) => (
        <button
          key={f.value}
          className={cx(styles.chip, active === f.value && styles.active)}
          onClick={() => onChange(f.value)}
        >
          {f.label}
        </button>
      ))}
    </div>
  );
}
