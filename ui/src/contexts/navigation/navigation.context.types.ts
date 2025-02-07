export interface MenuItem {
  id: string;
  icon?: string;
  title: string;
  to?: string;
  children?: MenuItem[];
}

export interface NavigationContextProps {
  navTree: MenuItem[];
  setNavTree: (navTree: MenuItem[]) => void;
}
