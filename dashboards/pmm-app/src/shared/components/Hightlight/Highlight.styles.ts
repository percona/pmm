import { css } from '@emotion/css';
import { createTheme, GrafanaTheme2 } from '@grafana/data';
import { ThemeMode } from './Highlight.types';

export const getStyles = (theme: GrafanaTheme2, mode?: ThemeMode) => {
  // v1 colours: TypeScript 4.8 cannot resolve the zod-inferred v2 colour types
  const { colors, isLight } = mode ? createTheme({ colors: { mode } }).v1 : theme.v1;
  const mainHighlightColor = isLight ? colors.text : '#e6e1dc';

  return {
    highlightWrapper: css`
      background-color: ${colors.bg1};
      border-color: ${colors.border2};
      color: ${colors.text};

      .hljs {
        display: block;
        overflow-x: auto;
        padding: 0.5em;
        background: transparent;
        color: ${mainHighlightColor};
      }
      .hljs-comment,
      .hljs-quote {
        color: #bc9458;
        font-style: italic;
      }
      .hljs-keyword,
      .hljs-selector-tag {
        color: #c26230;
      }
      .hljs-string,
      .hljs-number,
      .hljs-regexp,
      .hljs-variable,
      .hljs-template-variable {
        color: #a5c261;
      }
      .hljs-subst {
        color: #519f50;
      }
      .hljs-tag,
      .hljs-name {
        color: #e8bf6a;
      }
      .hljs-type {
        color: #da4939;
      }
      .hljs-symbol,
      .hljs-bullet,
      .hljs-built_in,
      .hljs-builtin-name,
      .hljs-attr,
      .hljs-link {
        color: #6d9cbe;
      }
      .hljs-params {
        color: #d0d0ff;
      }
      .hljs-attribute {
        color: #cda869;
      }
      .hljs-meta {
        color: #9b859d;
      }
      .hljs-title,
      .hljs-section {
        color: #ffc66d;
      }
      .hljs-addition {
        background-color: #144212;
        color: #e6e1dc;
        display: inline-block;
        width: 100%;
      }
      .hljs-deletion {
        background-color: #600;
        color: #e6e1dc;
        display: inline-block;
        width: 100%;
      }
      .hljs-selector-class {
        color: #9b703f;
      }
      .hljs-selector-id {
        color: #8b98ab;
      }
      .hljs-emphasis {
        font-style: italic;
      }
      .hljs-strong {
        font-weight: bold;
      }
      .hljs-link {
        text-decoration: underline;
      }
    `,
  };
};
