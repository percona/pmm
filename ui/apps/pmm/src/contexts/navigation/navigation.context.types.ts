import { NavItem } from 'lib/types';

export interface NavigationContextProps {
  navTree: NavItem[];
  navOpen: boolean;
  setNavOpen: (open: boolean) => void;
}
