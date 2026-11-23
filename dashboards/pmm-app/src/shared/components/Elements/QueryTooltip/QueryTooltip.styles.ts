import { css } from '@emotion/css';
import { GrafanaTheme2 } from '@grafana/data';

export const getStyles = ({ v1: { spacing, border } }: GrafanaTheme2, tooltipTheme: GrafanaTheme2) => {
  const { background, text } = tooltipTheme.components.tooltip;

  return {
    queryTooltip: (float: string) => css`
      max-width: 100%;
      background: ${background};

      .ant-tooltip-content {
        width: 600px;
        box-sizing: border-box;

        /* Popup shadow */
        box-shadow: 0px 2px 10px rgba(50, 116, 217, 0.25);
        border-radius: ${border.radius.sm};
        float: ${float};
      }

      .ant-tooltip-inner {
        background-color: transparent;
        border-radius: ${border.radius.sm};
        color: ${text};
        padding: 0;
      }

      .ant-tooltip-arrow:before {
        background-color: transparent;
      }
    `,
    queryTooltipDataWrapper: css`
      background-color: ${tooltipTheme.v1.colors.dashboardBg};
      padding: ${spacing.sm};
      border-radius: ${border.radius.md};
    `,
    queryId: css`
      margin: ${spacing.md};
      color: ${text};
    `,
  };
};
