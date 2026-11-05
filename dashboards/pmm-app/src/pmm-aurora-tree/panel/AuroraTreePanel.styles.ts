import { css } from '@emotion/css';
import { GrafanaTheme2 } from '@grafana/data';

// TypeScript 4.8 cannot instantiate the zod-inferred type of theme.colors (TS2589), so the colors
// used here are read through a narrow type.
interface ThemeColorsSubset {
  colors: {
    text: { secondary: string; link: string };
    border: { weak: string; medium: string };
    action: { hover: string };
  };
}

export const getStyles = (theme: GrafanaTheme2) => {
  const { text, border, action } = (theme as unknown as ThemeColorsSubset).colors;

  return {
    wrapper: css`
      display: flex;
      flex-direction: column;
      gap: ${theme.spacing(1)};
      height: 100%;
      overflow: auto;
    `,
    toolbar: css`
      display: flex;
      flex-wrap: wrap;
      align-items: center;
      gap: ${theme.spacing(1)};
    `,
    search: css`
      flex: 1 1 260px;
      max-width: 420px;
    `,
    count: css`
      margin-left: auto;
      color: ${text.secondary};
      white-space: nowrap;
    `,
    empty: css`
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      gap: ${theme.spacing(1)};
      flex: 1;
      min-height: ${theme.spacing(10)};
      color: ${text.secondary};
    `,
    table: css`
      width: 100%;
      border-collapse: collapse;
    `,
    header: css`
      padding: ${theme.spacing(1)};
      border-bottom: 1px solid ${border.weak};
      text-align: left;
      white-space: nowrap;
      font-weight: ${theme.typography.fontWeightMedium};
    `,
    sortButton: css`
      padding: 0;
      border: none;
      background: none;
      color: inherit;
      font: inherit;
      cursor: pointer;
      display: inline-flex;
      align-items: center;
      gap: ${theme.spacing(0.5)};
    `,
    expanderCell: css`
      width: ${theme.spacing(5)};
    `,
    row: css`
      border-bottom: 1px solid ${border.weak};

      &:hover {
        background: ${action.hover};
      }
    `,
    cell: css`
      padding: ${theme.spacing(1)};
    `,
    expandedCell: css`
      position: relative;
      padding: ${theme.spacing(1, 2, 2, 5)};
      border-bottom: 1px solid ${border.weak};

      &:before {
        content: '';
        position: absolute;
        top: 0;
        bottom: ${theme.spacing(2)};
        left: ${theme.spacing(2)};
        width: 1px;
        background: ${border.medium};
      }
    `,
    good: css`
      color: ${theme.visualization.getColorByName('green')};
    `,
    warning: css`
      color: ${theme.visualization.getColorByName('orange')};
    `,
    bad: css`
      color: ${theme.visualization.getColorByName('red')};
    `,
    unknownCluster: css`
      color: ${text.secondary};
      font-style: italic;
    `,
    links: css`
      display: inline-flex;
      gap: ${theme.spacing(1.5)};
      font-size: ${theme.typography.bodySmall.fontSize};
    `,
    link: css`
      color: ${text.link};

      &:hover {
        text-decoration: underline;
      }
    `,
  };
};
