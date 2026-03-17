import vscDarkPlus from 'react-syntax-highlighter/dist/esm/styles/prism/vsc-dark-plus';
import { Theme } from '@mui/material/styles';
import { primitives } from '@percona/percona-ui';
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
    base:
      language === 'text'
        ? theme.palette.text.primary
        : isLight
          ? primitives.brand.sky[600]
          : primitives.brand.sky[200],
    attrValue: isLight
      ? primitives.brand.aqua[700]
      : primitives.brand.aqua[300],
    string: isLight ? primitives.brand.aqua[700] : primitives.brand.aqua[300],
    number: theme.palette.text.primary,
    property: isLight
      ? primitives.brand.lavender[600]
      : primitives.brand.sky[200],
    function: isLight
      ? primitives.brand.lavender[600]
      : primitives.brand.lavender[200],
    operator: isLight ? primitives.brand.aqua[700] : primitives.brand.aqua[300],
    punctuation: theme.palette.text.secondary,
  };

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
