import type { BoxProps } from '@mui/material/Box';
import type { ElementType, ReactNode } from 'react';

export interface EmptyStateProps extends Omit<BoxProps, 'title' | 'children'> {
  /** What this place shows when it has content, in a few words. */
  title: ReactNode;
  /** Why it is empty right now, and what would fill it. */
  description?: ReactNode;
  /** A Peak illustration; rendered decorative (aria-hidden). */
  illustration?: ReactNode;
  /** One or two MUI Buttons or Links; the first is the way forward. */
  actions?: ReactNode;
  /** Element for the title; set it to fit the page outline. @default 'h3' */
  titleComponent?: ElementType;
  /** small: inside a card, table or section (py 4, illustration 48px); medium: a page or a whole section (py 6, illustration 64px). @default 'medium' */
  size?: 'small' | 'medium';
}
