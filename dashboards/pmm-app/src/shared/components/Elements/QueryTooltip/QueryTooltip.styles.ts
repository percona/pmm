import { css } from '@emotion/css';
import { createTheme, GrafanaTheme2 } from '@grafana/data';
import { ThemeMode } from 'shared/components/Hightlight/Highlight.types';

export const getStyles = ({ v1: { spacing, border } }: GrafanaTheme2, tooltipMode: ThemeMode) => {
  const tooltipTheme = createTheme({ colors: { mode: tooltipMode } });
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
