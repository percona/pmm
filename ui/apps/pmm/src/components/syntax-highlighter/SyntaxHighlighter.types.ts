import { SyntaxHighlighterProps as ReactSyntaxHighlighterProps } from 'react-syntax-highlighter';
import { CodeLanguage } from 'types/util.types';

export interface SyntaxHighlighterProps extends Omit<
  ReactSyntaxHighlighterProps,
  'children'
> {
  language: CodeLanguage;
  content: string;
  showCopyButton?: boolean;
  disableBorder?: boolean;
  /** When set, the code area scrolls inside the bordered box (e.g. "70vh") */
  maxHeight?: string | number;
}
