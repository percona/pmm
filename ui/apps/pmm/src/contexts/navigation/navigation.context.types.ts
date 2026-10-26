import { NavItem } from 'types/navigation.types';

export interface NavigationContextProps {
  navTree: NavItem[];
  navOpen: boolean;
  setNavOpen: (open: boolean) => void;
  // The ids of the apps starred from their headers, in the order they were starred.
  starredApps: string[];
  toggleStarredApp: (id: string) => void;
}
