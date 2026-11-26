import { NavItem } from 'types/navigation.types';

export interface NavigationContextProps {
  navTree: NavItem[];
  navOpen: boolean;
  setNavOpen: (open: boolean) => void;
}
