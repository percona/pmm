import type { BoxProps } from '@mui/material/Box';
import type { ElementType, ReactNode } from 'react';

export type EmptyStateIllustration =
  | 'nothing-found'
  | 'radar'
  | 'unknown-error'
  | 'real-time-database-off';

export type EmptyStateSize = 'small' | 'medium';

export interface EmptyStateProps extends Omit<BoxProps, 'title' | 'children'> {
  /** What this place shows when it has content, in a few words. */
  title: ReactNode;
  /** Why it is empty right now, and what would fill it. */
  description?: ReactNode;
  /** A Peak illustration, by name; sized by `size` and hidden from assistive technology. */
  illustration?: EmptyStateIllustration;
  /** One or two MUI Buttons or Links; the first is the way forward. */
  actions?: ReactNode;
  /**
   * Element for the title; set it to fit the page outline.
   * @default 'h3'
   */
  titleComponent?: ElementType;
  /**
   * small: inside a card, table or section; medium: a page or a whole section. Sets the padding and the illustration size.
   * @default 'medium'
   */
  size?: EmptyStateSize;
}
