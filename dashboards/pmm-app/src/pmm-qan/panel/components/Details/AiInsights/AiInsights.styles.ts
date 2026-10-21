import { css } from '@emotion/css';
import { colorManipulator, GrafanaTheme2 } from '@grafana/data';
import { getPmmTheme } from 'shared/components/helpers/getPmmTheme';

const CONTENT_WIDTH = 1040;

// The tokens used here; the plugin's TypeScript 4.8 cannot expand GrafanaTheme2's colors type (TS2589).
interface ThemeColors {
  text: Record<'primary' | 'secondary' | 'maxContrast' | 'link', string>;
  border: Record<'weak' | 'medium', string>;
  action: Record<'hover' | 'selected', string>;
  primary: Record<'main' | 'text', string>;
  background: Record<'canvas' | 'secondary', string>;
  error: { text: string };
  success: { text: string };
}

export const getStyles = (theme: GrafanaTheme2) => {
  const { alpha } = colorManipulator;
  const { colors } = theme as unknown as { colors: ThemeColors };
  const c = {
    background: getPmmTheme(theme.v1).table.backgroundColor,
    heading: `Poppins, ${theme.typography.fontFamily}`,
    mono: theme.typography.fontFamilyMonospace,
    text: colors.text.primary,
    textSecondary: colors.text.secondary,
    textStrong: colors.text.maxContrast,
    link: colors.text.link,
    error: colors.error.text,
    success: colors.success.text,
    borderWeak: colors.border.weak,
    borderMedium: colors.border.medium,
    hover: colors.action.hover,
    selected: colors.action.selected,
    primaryText: colors.primary.text,
    leadBackground: alpha(colors.primary.main, 0.08),
    leadBorder: alpha(colors.primary.main, 0.25),
    canvas: colors.background.canvas,
    headerCell: colors.background.secondary,
  };

  return {
    // The details grid sizes to its content; this keeps long code lines from widening it.
    container: css`
      contain: inline-size;
      max-width: ${CONTENT_WIDTH}px;
      padding: 0 16px 24px;
      min-height: 120px;
    `,
    loadingContainer: css`
      padding: 16px;
      min-height: 120px;
      display: flex;
      align-items: center;
      gap: 8px;
      color: ${c.textSecondary};
    `,
    message: css`
      white-space: pre-wrap;
      word-break: break-word;
    `,
    error: css`
      padding: 16px;
      color: ${c.error};
    `,
    // Stays in view while the answer scrolls under it.
    header: css`
      position: sticky;
      top: 0;
      z-index: 1;
      display: flex;
      align-items: center;
      justify-content: space-between;
      flex-wrap: wrap;
      gap: 4px 8px;
      padding: 10px 0;
      margin-bottom: 8px;
      background: ${c.background};
      border-bottom: 1px solid ${c.borderWeak};
    `,
    headerActions: css`
      display: flex;
      flex-wrap: wrap;
      align-items: center;
      gap: 2px;
    `,
    cachedAt: css`
      font-size: 12px;
      color: ${c.textSecondary};
    `,
    ticketCreated: css`
      font-size: 12px;
      color: ${c.success};
      margin-bottom: 8px;
    `,
    // An answer reads as a document: a highlighted lead, numbered steps, and headings in Poppins.
    analysis: css`
      font-size: 14px;
      line-height: 1.65;
      color: ${c.text};
      overflow-wrap: anywhere;

      h1, h2, h3, h4, h5, h6 {
        font-family: ${c.heading};
        font-weight: 600;
        line-height: 1.4;
        color: ${c.textStrong};
        margin: 24px 0 8px;
      }
      h1 { font-size: 20px; }
      h2 { font-size: 18px; }
      h3, h4, h5, h6 { font-size: 15px; }
      & > :first-child { margin-top: 8px; }

      p { margin: 0 0 12px; }
      strong { color: ${c.textStrong}; font-weight: 600; }
      a { color: ${c.link}; }

      /* The lead: the opening paragraph, or the one under an opening heading such as Summary. */
      & > p:first-child,
      & > h1:first-child + p,
      & > h2:first-child + p,
      & > h3:first-child + p {
        padding: 16px 18px;
        margin-bottom: 20px;
        border-radius: 14px;
        background: ${c.leadBackground};
        border: 1px solid ${c.leadBorder};
      }

      ul { padding-left: 22px; margin: 8px 0 12px; }
      ul > li { margin: 6px 0; }
      ol { list-style: none; padding-left: 0; margin: 12px 0; }
      ol > li { position: relative; padding-left: 40px; margin: 14px 0; }
      ol > li::before {
        content: attr(data-n);
        position: absolute;
        left: 0;
        top: 0;
        width: 26px;
        height: 26px;
        border-radius: 50%;
        background: ${c.selected};
        color: ${c.primaryText};
        font-size: 13px;
        font-weight: 500;
        line-height: 26px;
        text-align: center;
      }

      blockquote {
        margin: 12px 0;
        padding-left: 16px;
        color: ${c.textSecondary};
        border-left: 3px solid ${c.borderMedium};
      }

      hr {
        border: 0;
        border-top: 1px solid ${c.borderWeak};
        margin: 20px 0;
      }

      /* Grafana outlines every <code>; inline code here is a quiet tint. */
      code {
        font-family: ${c.mono};
        font-size: 0.86em;
        padding: 1px 5px;
        border: 0;
        border-radius: 4px;
        background: ${c.hover};
        color: ${c.text};
        white-space: normal;
      }

      .ai-code {
        margin: 12px 0 16px;
        border: 1px solid ${c.borderWeak};
        border-radius: 10px;
        background: ${c.canvas};
        overflow: hidden;
      }
      .ai-code-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        min-height: 34px;
        padding: 0 6px 0 14px;
        border-bottom: 1px solid ${c.borderWeak};
        font-size: 12px;
        color: ${c.textSecondary};
      }
      .ai-code-header button {
        padding: 2px 8px;
        border: 0;
        border-radius: 4px;
        background: transparent;
        color: ${c.textSecondary};
        font-size: 12px;
        cursor: pointer;
      }
      .ai-code-header button:hover,
      .ai-code-header button:focus-visible {
        background: ${c.hover};
        color: ${c.text};
      }
      .ai-code-header button[data-copied] { color: ${c.success}; }
      .ai-code pre {
        margin: 0;
        padding: 12px 14px;
        border: 0;
        border-radius: 0;
        background: transparent;
        overflow-x: auto;
        line-height: 1.6;
      }
      .ai-code pre code {
        display: block;
        padding: 0;
        background: transparent;
        white-space: pre;
        font-size: 13px;
      }

      .ai-table {
        margin: 12px 0 16px;
        overflow-x: auto;
        border: 1px solid ${c.borderWeak};
        border-radius: 10px;
      }
      table {
        width: 100%;
        border-collapse: collapse;
        font-size: 13px;
        font-variant-numeric: tabular-nums;
      }
      th, td {
        padding: 7px 12px;
        text-align: left;
        vertical-align: top;
        border-bottom: 1px solid ${c.borderWeak};
      }
      th {
        font-weight: 500;
        color: ${c.textSecondary};
        background: ${c.headerCell};
        white-space: nowrap;
      }
      th[align='right'], td[align='right'] { text-align: right; }
      th[align='center'], td[align='center'] { text-align: center; }
      tbody tr:last-child td { border-bottom: 0; }
      tbody tr:hover td { background: ${c.hover}; }
    `,
  };
};
