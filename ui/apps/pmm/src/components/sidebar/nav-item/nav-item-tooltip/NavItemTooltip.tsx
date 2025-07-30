import Tooltip from '@mui/material/Tooltip';
import { NavItem } from 'lib/types';
import { FC, ReactElement } from 'react';

interface Props {
  children: ReactElement;
  item: NavItem;
  drawerOpen: boolean;
}

const NavItemTooltip: FC<Props> = ({ children, item, drawerOpen }) => {
  if (drawerOpen) {
    return children;
  }

  return (
    <Tooltip title={item.text} placement="right" enterDelay={500} arrow>
      {children}
    </Tooltip>
  );
};

export default NavItemTooltip;
