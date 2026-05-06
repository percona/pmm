import type { ListItemButtonProps } from '@mui/material/ListItemButton';
import type { ElementType, ReactNode } from 'react';

export type NavItemDotColor = 'success' | 'info' | 'warning' | 'error';

export interface NavItemProps extends Omit<ListItemButtonProps, 'children' | 'disableGutters'> {
  text: string;
  secondaryText?: string;
  icon?: ReactNode;
  badge?: ReactNode;
  showDot?: boolean;
  dotColor?: NavItemDotColor;
  component?: ElementType;
}
