import type { ReactNode } from 'react';
import type { BoxProps } from '@mui/material/Box';

export type CodeBlockProps = Omit<BoxProps<'pre'>, 'component' | 'children'> & {
  /** Text or elements shown in the block; a plain string can be highlighted and copied. */
  content: ReactNode;
  /** Shows a copy button in the top-right corner. */
  copyable?: boolean;
  /** Makes the copy button a labeled button ("Copy code") instead of a bare icon. */
  showCopyButtonText?: boolean;
  /** Text to copy and highlight when it differs from `content`, for example when `content` holds elements. */
  value?: string;
  /** Language for syntax highlighting, such as bash, yaml, or json; without it the text is shown as-is. */
  language?: string;
  /** Wraps long lines instead of scrolling horizontally. */
  wrap?: boolean;
};
