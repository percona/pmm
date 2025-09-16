import { IconName } from 'components/icon/Icon.types';
import { SvgIconComponent } from './util.types';

export interface NavItem {
  id: string;
  text?: string;
  secondaryText?: string;
  icon?: IconName | SvgIconComponent;
  url?: string;
  children?: NavItem[];
  isActive?: boolean;
  target?: HTMLAnchorElement['target'];
  isDivider?: boolean;
  onClick?: () => void;
  hidden?: boolean;
  badge?: string;
}
