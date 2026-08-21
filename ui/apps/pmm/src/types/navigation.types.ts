import type { IconName } from 'components/icon/Icon.types';
import type { SvgIconComponent } from './util.types';
import type { ChipProps } from '@mui/material/Chip';

export interface NavItem {
  id: string;
  text?: string;
  secondaryText?: string;
  icon?: IconName | SvgIconComponent | React.ReactElement | React.ComponentType;
  url?: string;
  children?: NavItem[];
  isActive?: boolean;
  target?: HTMLAnchorElement['target'];
  onClick?: () => void;
  hidden?: boolean;
  badge?: ChipProps | React.ReactElement;
  badgeAlwaysVisible?: boolean;
  matches?: string[];
  type?: 'menu-item' | 'menu-text' | 'menu-divider';
}
