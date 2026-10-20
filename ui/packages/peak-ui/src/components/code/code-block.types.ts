import type { ReactNode } from 'react';
import type { BoxProps } from '@mui/material/Box';

export type CodeBlockProps = Omit<BoxProps<'pre'>, 'component' | 'children'> & {
  content: ReactNode;
  copyable?: boolean;
  showCopyButtonText?: boolean;
  value?: string;
  language?: string;
  wrap?: boolean;
};
