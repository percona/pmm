import { css } from '@emotion/css';

const s = {
  bar: css`
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 8px 10px;
    padding: 8px 14px;
    background: linear-gradient(180deg, #16162a 0%, #12121f 100%);
    border-bottom: 1px solid #3a4a6a;
    flex-shrink: 0;
  `,
  label: css`
    font-size: 11px;
    color: #b8b8d8;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    margin-right: 4px;
  `,
  hint: css`
    font-size: 10px;
    color: #6a6a8a;
    flex-basis: 100%;
    margin-top: -2px;
  `,
  chip: (active: boolean) => css`
    padding: 4px 10px;
    border-radius: 6px;
    font-size: 11px;
    font-weight: 600;
    cursor: pointer;
    border: 1px solid ${active ? '#5a8fd0' : '#4a4a6a'};
    background: ${active ? 'rgba(90, 143, 208, 0.22)' : '#1e1e32'};
    color: ${active ? '#e8eef8' : '#a0a0c0'};
    transition: border-color 0.15s, background 0.15s;
    &:hover {
      border-color: #7aa3e0;
      color: #fff;
    }
  `,
};

interface Props {
  groupByPod: boolean;
  hideWeakEdges: boolean;
  onGroupByPodChange: (next: boolean) => void;
  onHideWeakEdgesChange: (next: boolean) => void;
}

export function ViewOptionsBar({
  groupByPod,
  hideWeakEdges,
  onGroupByPodChange,
  onHideWeakEdgesChange,
}: Props) {
  return (
    <div className={s.bar}>
      <span className={s.label}>View</span>
      <button
        type="button"
        className={s.chip(groupByPod)}
        onClick={() => onGroupByPodChange(!groupByPod)}
        title="Collapse /k8s/ns/pod/container to one node per pod"
      >
        Group by pod
      </button>
      <button
        type="button"
        className={s.chip(hideWeakEdges)}
        onClick={() => onHideWeakEdgesChange(!hideWeakEdges)}
        title="Hide healthy edges with very low L7 request rate (see panel Weak edge max RPS)"
      >
        Hide weak edges
      </button>
      <span className={s.hint}>
        Panel options only set the initial defaults when the dashboard loads.
      </span>
    </div>
  );
}
