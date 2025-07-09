import { IconName } from 'components/icon/Icon.types';

export interface NavItem {
  id: string;
  text?: string;
  icon?: IconName;
  url?: string;
  children?: NavItem[];
  isActive?: boolean;
  target?: HTMLAnchorElement['target'];
  isDivider?: boolean;
  onClick?: () => void;
  hidden?: boolean;
}
