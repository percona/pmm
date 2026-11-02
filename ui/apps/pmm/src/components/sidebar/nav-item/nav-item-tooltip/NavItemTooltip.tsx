import Tooltip from '@mui/material/Tooltip';
import { NavItem } from 'types/navigation.types';
import { FC, ReactElement } from 'react';

interface Props {
  children: ReactElement;
  item: NavItem;
  drawerOpen: boolean;
  level: number;
}

const NavItemTooltip: FC<Props> = ({ children, item, level, drawerOpen }) => {
  if (level !== 0) {
    return children;
  }

  // Rendered in both drawer states, so toggling the drawer never remounts the
  // item and its shape can animate. An empty title keeps it shut while open.
  return (
    <Tooltip
      title={drawerOpen ? '' : item.text}
      placement="right"
      enterDelay={500}
      arrow
    >
      {children}
    </Tooltip>
  );
};

export default NavItemTooltip;
