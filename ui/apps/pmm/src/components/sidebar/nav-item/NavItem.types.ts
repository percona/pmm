import { NavItem } from 'lib/types';

export interface NavItemProps {
  item: NavItem;
  level?: number;
  drawerOpen: boolean;
}
