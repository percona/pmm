import { NavItem } from 'types/navigation.types';

export interface NavItemProps {
  item: NavItem;
  level?: number;
  drawerOpen: boolean;
}
