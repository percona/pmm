import vscDarkPlus from 'react-syntax-highlighter/dist/esm/styles/prism/vsc-dark-plus';
import { Theme } from '@mui/material/styles';
import { PEAK_DARK_THEME, PEAK_LIGHT_THEME } from '@pmm/shared';
import { CodeLanguage } from 'types/util.types';

export const getSyntaxHighlighterStyle = (
  theme: Theme,
  language: CodeLanguage,
  showCopyButton = false
) => {
  const isLight = theme.palette.mode === 'light';

  const tokens = {
    fontFamily: 'Roboto Mono, monospace',
    background: 'transparent',
    base: isLight ? PEAK_LIGHT_THEME.text.sky : PEAK_DARK_THEME.text.sky,
    attrValue: isLight ? PEAK_LIGHT_THEME.text.aqua : PEAK_DARK_THEME.text.aqua,
    string: isLight ? PEAK_LIGHT_THEME.text.aqua : PEAK_DARK_THEME.text.aqua,
    number: isLight
      ? PEAK_LIGHT_THEME.text.primary
      : PEAK_DARK_THEME.text.primary,
    property: isLight
      ? PEAK_LIGHT_THEME.text.lavender
      : PEAK_DARK_THEME.text.sky,
    function: isLight
      ? PEAK_LIGHT_THEME.text.lavender
      : PEAK_DARK_THEME.text.lavender,
    operator: isLight ? PEAK_LIGHT_THEME.text.aqua : PEAK_DARK_THEME.text.aqua,
    punctuation: theme.palette.text.secondary,
  };

  if (language === 'text') {
    tokens.base = theme.palette.text.primary;
  }

  // Define your custom styles to override the base VSC Dark Plus colors
  const customStyle = {
    ...vscDarkPlus,
    'pre[class*="language-"]': {
      margin: 0,
      paddingLeft: theme.spacing(2),
      paddingRight: theme.spacing(showCopyButton ? 3.5 : 2),
      paddingTop: theme.spacing(1),
      paddingBottom: theme.spacing(1),
      background: tokens.background,
      fontFamily: tokens.fontFamily,
      overflow: 'hidden',
      textOverflow: 'ellipsis',
      whiteSpace: 'nowrap',
    },
    'code[class*="language-"]': {
      ...vscDarkPlus['code[class*="language-"]'],
      background: tokens.background,
      color: tokens.base,
      fontFamily: tokens.fontFamily,
    },
    'attr-value': {
      color: tokens.attrValue,
    },
    function: {
      color: tokens.function,
    },
    property: {
      color: tokens.property,
    },
    string: {
      color: tokens.string,
    },
    number: {
      color: tokens.number,
    },
    operator: {
      color: tokens.operator,
    },
    punctuation: {
      color: tokens.punctuation,
    },
  };

  return customStyle;
};
