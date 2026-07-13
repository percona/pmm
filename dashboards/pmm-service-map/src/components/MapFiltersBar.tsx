import { css } from '@emotion/css';

const s = {
  bar: css`
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 6px 12px;
    padding: 6px 12px;
    background: linear-gradient(180deg, #16162a 0%, #12121f 100%);
    border-bottom: 1px solid #3a4a6a;
    flex-shrink: 0;
  `,
  label: css`
    font-size: 10px;
    color: #b8b8d8;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    margin-right: 2px;
  `,
  hint: css`
    font-size: 9px;
    color: #5a5a7a;
    flex: 1 1 100%;
    line-height: 1.25;
  `,
  chip: (active: boolean) => css`
    padding: 3px 8px;
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
  sep: css`
    width: 1px;
    height: 18px;
    background: #3a4a6a;
    margin: 0 4px;
    flex-shrink: 0;
  `,
  podInput: css`
    min-width: 140px;
    max-width: 280px;
    flex: 1 1 160px;
    padding: 4px 8px;
    font-size: 11px;
    border-radius: 6px;
    border: 1px solid #4a4a6a;
    background: #1e1e32;
    color: #e0e0e0;
    &::placeholder {
      color: #6a6a8a;
    }
  `,
};

interface Props {
  groupByPod: boolean;
  hideWeakEdges: boolean;
  onGroupByPodChange: (next: boolean) => void;
  onHideWeakEdgesChange: (next: boolean) => void;
  namespaces: string[];
  nsSelected: Set<string>;
  onNsChange: (next: Set<string>) => void;
  podNameFilter: string;
  onPodNameFilterChange: (next: string) => void;
}

export function MapFiltersBar({
  groupByPod,
  hideWeakEdges,
  onGroupByPodChange,
  onHideWeakEdgesChange,
  namespaces,
  nsSelected,
  onNsChange,
  podNameFilter,
  onPodNameFilterChange,
}: Props) {
  const allActive = nsSelected.size === 0;

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

      <span className={s.sep} aria-hidden />

      <span className={s.label}>Namespaces</span>
      <button type="button" className={s.chip(allActive)} onClick={() => onNsChange(new Set())}>
        All
      </button>
      {namespaces.map((ns) => {
        const chipActive = !allActive && nsSelected.has(ns);
        return (
          <button
            key={ns}
            type="button"
            className={s.chip(chipActive)}
            onClick={() => {
              if (allActive) {
                onNsChange(new Set([ns]));
              } else {
                const next = new Set(nsSelected);
                if (next.has(ns)) {
                  next.delete(ns);
                } else {
                  next.add(ns);
                }
                onNsChange(next);
              }
            }}
            title={ns}
          >
            {ns}
          </button>
        );
      })}

      <span className={s.sep} aria-hidden />

      <span className={s.label}>Pod</span>
      <input
        type="search"
        className={s.podInput}
        value={podNameFilter}
        onChange={(e) => onPodNameFilterChange(e.target.value)}
        placeholder="Filter by name…"
        title="Matches workload id or label; keeps that workload and every neighbor connected by an edge"
        aria-label="Filter graph by pod name substring"
      />

      <span className={s.hint}>
        Panel options set defaults on load. Pod filter shows workloads whose id or label contains your text,
        plus any service linked by an incoming or outgoing edge (even if that neighbor does not match).
      </span>
    </div>
  );
}
