import { ReactNode } from 'react';
import { NavItem } from 'types/navigation.types';

export interface NavItemProps {
  item: NavItem;
  level?: number;
  drawerOpen: boolean;
  activeItem: NavItem;
  onClick?: () => void;
  // A control on a section heading's label line, such as Starred's edit
  // button. Laid over the heading so it never changes the heading's height.
  headingAccessory?: ReactNode;
  // A button over an entry's rounded end, such as a reorder handle.
  trailingControl?: ReactNode;
  // Keeps the entry in place but stops it navigating, while its list is
  // being edited.
  disableLink?: boolean;
  // Listed in a rail group's panel, so drawn as a menu row.
  inFlyout?: boolean;
}
