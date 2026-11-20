import { css } from '@emotion/css';
import { GrafanaTheme2 } from '@grafana/data';

export const getStyles = ({ v1: { colors } }: GrafanaTheme2) => ({
  highlightWrapper: css`
    background-color: ${colors.bg1};
    border-color: ${colors.border2};

    code {
      display: block;
      overflow-x: auto;
      padding: 0.5em;
      background: transparent;
    }
  `,
});
