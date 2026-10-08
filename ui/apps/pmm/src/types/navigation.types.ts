import { IconName } from 'components/icon/Icon.types';
import { SvgIconComponent } from './util.types';
import { ChipProps } from '@mui/material/Chip';

// A small trailing icon button beside an entry or a section heading. It is
// rendered outside the entry's own link so the two stay separate targets.
export interface NavItemAction {
  label: string;
  url: string;
  icon?: SvgIconComponent;
}

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
  type?: 'menu-item' | 'menu-text' | 'menu-divider' | 'menu-section';
  action?: NavItemAction;
  // A pinned entry shows a reorder handle.
  pinned?: boolean;
}
