import { css } from '@emotion/css';
import { GrafanaTheme2 } from '@grafana/data';

// TypeScript 4.8 cannot instantiate the zod-inferred type of theme.colors (TS2589), so the two text
// colors are read through a narrow type.
interface ThemeTextColors {
  colors: { text: { secondary: string; link: string } };
}

export const getStyles = (theme: GrafanaTheme2) => {
  const { text } = (theme as unknown as ThemeTextColors).colors;

  return {
    wrapper: css`
      height: 100%;
      overflow: auto;
    `,
    empty: css`
      display: flex;
      align-items: center;
      justify-content: center;
      height: 100%;
      color: ${text.secondary};
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
    link: css`
      color: ${text.link};

      &:hover {
        text-decoration: underline;
      }
    `,
  };
};
