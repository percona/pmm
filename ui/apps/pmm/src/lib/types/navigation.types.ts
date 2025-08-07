export interface NavItem {
  id: string;
  text: string;
  icon?: string;
  url: string;
  children?: NavItem[];
  isActive?: boolean;
  target?: HTMLAnchorElement['target'];
}
