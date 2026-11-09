import { css } from '@emotion/css';
import type { GrafanaTheme2 } from '@grafana/data';

export const getStyles = ({ v1: { palette } }: GrafanaTheme2) => ({
  wrapper: css`
    padding: 0.5em;
  `,
  header: css`
    border-bottom: 1px solid ${palette.gray2};
    padding: 0em 0.5em;
    margin-bottom: 0.5em;
  `,
  body: css`
    padding: 0em 0.5em;
    font-weight: normal;
  `,
});
