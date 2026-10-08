import type { ReactNode } from 'react';
import type { BoxProps } from '@mui/material/Box';

export type CodeBlockProps = Omit<BoxProps<'pre'>, 'component' | 'children'> & {
  /** Text or elements shown in the block; a plain string can be highlighted and copied. */
  content: ReactNode;
  /** Shows a copy button in the top-right corner. */
  copyable?: boolean;
  /** Makes the copy button a labeled button, with text ("Copy code"), instead of a bare icon. */
  showCopyButtonText?: boolean;
  /** Text to copy when it differs from `content`, such as when `content` holds elements; with `language` set it is shown instead of `content`. */
  value?: string;
  /** Language for syntax highlighting, such as sql, yaml, or json; without a supported one the text is shown in a single plain color. */
  language?: string;
  /** Wraps long lines instead of forcing users to scroll horizontally. */
  wrap?: boolean;
};
