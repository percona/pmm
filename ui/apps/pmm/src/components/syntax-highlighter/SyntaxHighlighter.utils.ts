import vscDarkPlus from 'react-syntax-highlighter/dist/esm/styles/prism/vsc-dark-plus';
import { Theme } from '@mui/material/styles';
import { semanticTokensLight, semanticTokensDark } from '@percona/percona-ui';
import { CodeLanguage } from 'types/util.types';

export const getSyntaxHighlighterStyle = (
  theme: Theme,
  language: CodeLanguage,
  showCopyButton = false
) => {
  const accents =
    theme.palette.mode === 'light'
      ? semanticTokensLight.text
      : semanticTokensDark.text;

  const tokens = {
    fontFamily: 'Roboto Mono, monospace',
    background: 'transparent',
    base:
      language === 'text' ? theme.palette.text.primary : accents.accent1,
    attrValue: accents.accent3,
    string: accents.accent3,
    number: theme.palette.text.primary,
    property: accents.accent2,
    function: accents.accent2,
    operator: accents.accent3,
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
