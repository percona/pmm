import type { ListItemButtonProps } from '@mui/material/ListItemButton';
import type { ElementType, ReactNode } from 'react';

export type NavItemDotColor = 'success' | 'info' | 'warning' | 'error';

export interface NavItemProps extends Omit<ListItemButtonProps, 'children' | 'disableGutters'> {
  /** Label text of the item. */
  text: string;
  /** Smaller text under the label, cut with an ellipsis when it's too long. */
  secondaryText?: string;
  /** Icon shown before the text label, to the left; without one, the label starts where sibling rows' icons start. */
  icon?: ReactNode;
  /** Content shown at the end of the row, such as a count chip or New/Preview labelling. */
  badge?: ReactNode;
  /** Shows a small status dot on the icon's corner; needs an `icon` to show up. */
  showDot?: boolean;
  /** Color of the status dot. Defaults to warning. */
  dotColor?: NavItemDotColor;
  /** Element to render as, for example a router link so the row navigates. */
  component?: ElementType;
}
