import { IconName } from 'components/icon/Icon.types';
import { SvgIconComponent } from './util.types';
import { ChipProps } from '@mui/material/Chip';

// A small add button beside an entry, outside its link so the two stay
// separate targets.
export interface NavItemAction {
  label: string;
  url: string;
}

export interface NavItem {
  id: string;
  text?: string;
  secondaryText?: string;
  icon?: IconName | SvgIconComponent | React.ReactElement | React.ComponentType;
  url?: string;
  children?: NavItem[];
  isActive?: boolean;
  // The child a collapsible parent links to; defaults to the first child.
  isDefault?: boolean;
  target?: HTMLAnchorElement['target'];
  onClick?: () => void;
  hidden?: boolean;
  badge?: ChipProps | React.ReactElement;
  badgeAlwaysVisible?: boolean;
  matches?: string[];
  type?: 'menu-item' | 'menu-text' | 'menu-section';
  action?: NavItemAction;
}
