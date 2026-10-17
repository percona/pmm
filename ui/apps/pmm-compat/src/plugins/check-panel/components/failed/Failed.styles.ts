import { css } from '@emotion/css';
import type { GrafanaTheme2 } from '@grafana/data';

export const getStyles = ({ v1: { palette, colors } }: GrafanaTheme2) => ({
  failedDiv: css`
    margin-right: 0.5em;
    font-size: 30px;
  `,
  green: css`
    color: ${palette.greenBase};
  `,
  empty: css`
    text-align: center;
  `,
  link: css`
    color: ${colors.linkExternal};
    &:hover {
      color: ${colors.textBlue};
    }
  `,
  critical: css`
    color: ${palette.red};
  `,
  error: css`
    color: ${palette.orange};
  `,
  warning: css`
    color: ${palette.yellow};
  `,
  notice: css`
    color: ${palette.blue80};
  `,
});
