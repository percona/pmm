import Tooltip from '@mui/material/Tooltip';
import {
  DRAWER_CLOSED_WIDTH,
  DRAWER_WIDTH,
} from 'components/sidebar/drawer/Drawer.constants';
import { NavItem } from 'types/navigation.types';
import { FC, ReactElement } from 'react';

interface Props {
  children: ReactElement;
  item: NavItem;
  drawerOpen: boolean;
  level: number;
}

const NavItemTooltip: FC<Props> = ({ children, item, level, drawerOpen }) => {
  if (drawerOpen || level !== 0) {
    return children;
  }

  return (
    <Tooltip
      title={item.text}
      placement="right"
      enterDelay={500}
      arrow
      slotProps={{
        popper: {
          modifiers: [
            {
              name: 'offset',
              options: {
                offset: [0, DRAWER_CLOSED_WIDTH - DRAWER_WIDTH],
              },
            },
          ],
        },
      }}
    >
      {children}
    </Tooltip>
  );
};

export default NavItemTooltip;
