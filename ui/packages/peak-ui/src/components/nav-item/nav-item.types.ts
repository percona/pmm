import type { ListItemButtonProps } from '@mui/material/ListItemButton';
import type { ElementType, ReactNode } from 'react';

export type NavItemDotColor = 'success' | 'info' | 'warning' | 'error';

export interface NavItemProps extends Omit<ListItemButtonProps, 'children' | 'disableGutters'> {
  /** Label of the row. */
  text: string;
  /** Smaller text under the label, cut with an ellipsis when too long. */
  secondaryText?: string;
  /** Icon shown before the label; without one the label still lines up with rows that have icons. */
  icon?: ReactNode;
  /** Content shown at the end of the row, such as a count chip. */
  badge?: ReactNode;
  /** Shows a small status dot on the icon's corner; needs an `icon`. */
  showDot?: boolean;
  /** Color of the status dot. Defaults to warning. */
  dotColor?: NavItemDotColor;
  /** Element to render as, for example a router link so the row navigates. */
  component?: ElementType;
}
