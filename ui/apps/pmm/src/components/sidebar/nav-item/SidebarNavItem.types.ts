import { NavItem } from 'types/navigation.types';

export interface NavItemProps {
  item: NavItem;
  level?: number;
  drawerOpen: boolean;
  activeItem?: NavItem;
  onClick?: () => void;
  // Listed in a rail group's panel, so drawn as a menu row.
  inFlyout?: boolean;
}
