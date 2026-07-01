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
  namespaces: string[];
  /** Empty set = show all namespaces */
  selected: Set<string>;
  onChange: (next: Set<string>) => void;
}

export function NamespaceFilter({ namespaces, selected, onChange }: Props) {
  const allActive = selected.size === 0;

  return (
    <div className={s.bar}>
      <span className={s.label}>Namespaces</span>
      <button type="button" className={s.chip(allActive)} onClick={() => onChange(new Set())}>
        All
      </button>
      {namespaces.map((ns) => {
        const chipActive = !allActive && selected.has(ns);
        return (
          <button
            key={ns}
            type="button"
            className={s.chip(chipActive)}
            onClick={() => {
              if (allActive) {
                onChange(new Set([ns]));
              } else {
                const next = new Set(selected);
                if (next.has(ns)) {
                  next.delete(ns);
                } else {
                  next.add(ns);
                }
                onChange(next);
              }
            }}
            title={ns}
          >
            {ns}
          </button>
        );
      })}
      <span className={s.hint}>
        {allActive
          ? 'Showing every namespace. Click a name to filter; add more for multi-select.'
          : `${selected.size} namespace(s) selected — click All to reset.`}
      </span>
    </div>
  );
}
