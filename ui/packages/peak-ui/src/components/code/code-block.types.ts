import type { ReactNode } from 'react';
import type { BoxProps } from '@mui/material/Box';
import type { PrismTheme } from 'prism-react-renderer';

export type CodeColorScheme =
  | 'github'
  | 'vsLight'
  | 'vsDark'
  | 'oneLight'
  | 'oneDark'
  | 'dracula'
  | 'nightOwl'
  | 'nightOwlLight'
  | 'oceanicNext'
  | 'palenight'
  | 'okaidia';

export type CodeBlockProps = Omit<BoxProps<'pre'>, 'component' | 'children'> & {
  children: ReactNode;
  copyable?: boolean;
  showCopyButtonText?: boolean;
  value?: string;
  language?: string;
  colorScheme?: CodeColorScheme | PrismTheme;
};
