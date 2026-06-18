import type { ReactNode } from 'react';
import type { BoxProps } from '@mui/material/Box';
import type { PrismTheme } from 'prism-react-renderer';

export type CodeColorScheme =
  | 'dracula'
  | 'duotoneDark'
  | 'duotoneLight'
  | 'github'
  | 'gruvboxMaterialDark'
  | 'gruvboxMaterialLight'
  | 'jettwaveDark'
  | 'jettwaveLight'
  | 'nightOwl'
  | 'nightOwlLight'
  | 'oceanicNext'
  | 'okaidia'
  | 'oneDark'
  | 'oneLight'
  | 'palenight'
  | 'shadesOfPurple'
  | 'synthwave84'
  | 'ultramin'
  | 'vsDark'
  | 'vsLight';

export type CodeBlockProps = Omit<BoxProps<'pre'>, 'component' | 'children'> & {
  content: ReactNode;
  copyable?: boolean;
  showCopyButtonText?: boolean;
  value?: string;
  language?: string;
  colorScheme?: CodeColorScheme | PrismTheme;
  wrap?: boolean;
};
