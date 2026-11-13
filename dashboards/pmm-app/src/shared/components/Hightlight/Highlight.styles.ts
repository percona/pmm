import { css } from '@emotion/css';
import { createTheme, GrafanaTheme2 } from '@grafana/data';
import { ThemeMode } from './Highlight.types';

const darkPalette = {
  comment: '#bc9458',
  keyword: '#c26230',
  string: '#a5c261',
  subst: '#519f50',
  tag: '#e8bf6a',
  type: '#da4939',
  builtIn: '#6d9cbe',
  params: '#d0d0ff',
  attribute: '#cda869',
  meta: '#9b859d',
  title: '#ffc66d',
  addition: '#e6e1dc',
  additionBg: '#144212',
  deletion: '#e6e1dc',
  deletionBg: '#600',
  selectorClass: '#9b703f',
  selectorId: '#8b98ab',
};

// GitHub light colours, legible on a white background
const lightPalette: typeof darkPalette = {
  comment: '#6a737d',
  keyword: '#d73a49',
  string: '#032f62',
  subst: '#24292e',
  tag: '#22863a',
  type: '#d73a49',
  builtIn: '#e36209',
  params: '#24292e',
  attribute: '#005cc5',
  meta: '#005cc5',
  title: '#6f42c1',
  addition: '#22863a',
  additionBg: '#f0fff4',
  deletion: '#b31d28',
  deletionBg: '#ffeef0',
  selectorClass: '#005cc5',
  selectorId: '#005cc5',
};

export const getStyles = (theme: GrafanaTheme2, mode?: ThemeMode) => {
  // v1 colours: TypeScript 4.8 cannot resolve the zod-inferred v2 colour types
  const { colors, isLight } = mode ? createTheme({ colors: { mode } }).v1 : theme.v1;
  const mainHighlightColor = isLight ? colors.text : '#e6e1dc';
  const palette = isLight ? lightPalette : darkPalette;

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
        color: ${palette.comment};
        font-style: italic;
      }
      .hljs-keyword,
      .hljs-selector-tag {
        color: ${palette.keyword};
      }
      .hljs-string,
      .hljs-number,
      .hljs-regexp,
      .hljs-variable,
      .hljs-template-variable {
        color: ${palette.string};
      }
      .hljs-subst {
        color: ${palette.subst};
      }
      .hljs-tag,
      .hljs-name {
        color: ${palette.tag};
      }
      .hljs-type {
        color: ${palette.type};
      }
      .hljs-symbol,
      .hljs-bullet,
      .hljs-built_in,
      .hljs-builtin-name,
      .hljs-attr,
      .hljs-link {
        color: ${palette.builtIn};
      }
      .hljs-params {
        color: ${palette.params};
      }
      .hljs-attribute {
        color: ${palette.attribute};
      }
      .hljs-meta {
        color: ${palette.meta};
      }
      .hljs-title,
      .hljs-section {
        color: ${palette.title};
      }
      .hljs-addition {
        background-color: ${palette.additionBg};
        color: ${palette.addition};
        display: inline-block;
        width: 100%;
      }
      .hljs-deletion {
        background-color: ${palette.deletionBg};
        color: ${palette.deletion};
        display: inline-block;
        width: 100%;
      }
      .hljs-selector-class {
        color: ${palette.selectorClass};
      }
      .hljs-selector-id {
        color: ${palette.selectorId};
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
