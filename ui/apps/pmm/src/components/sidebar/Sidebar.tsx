import { FC, useCallback, useEffect, useState } from 'react';
import { useNavigation } from 'contexts/navigation';
import { NavigationHeading } from './nav-heading';
import { Drawer } from './drawer';
import { SidebarNavItem } from './nav-item';
import List from '@mui/material/List';
import useMediaQuery from '@mui/material/useMediaQuery';
import { useTheme } from '@mui/material/styles';
import { findActiveNavItem } from 'utils/navigation.utils';
import { useLocation } from 'react-router-dom';
import { NavItem as NavItemType } from 'types/navigation.types';

export const Sidebar: FC = () => {
  const { navTree, navOpen, setNavOpen } = useNavigation();
  const [activeItem, setActiveItem] = useState<NavItemType>(navTree[0]);
  const [animating, setAnimating] = useState(false);
  const location = useLocation();
  const theme = useTheme();
  const isNarrow = useMediaQuery(theme.breakpoints.down('md'));

  const toggleSidebar = useCallback(() => {
    setNavOpen(!navOpen);

    setAnimating(true);

    const timeoutId = setTimeout(() => setAnimating(false), 2000);

    return () => {
      clearTimeout(timeoutId);
    };
  }, [navOpen, setNavOpen]);

  useEffect(() => {
    const activeItem = findActiveNavItem(navTree, location.pathname);

    // keep previous item active if there isn't a match
    if (activeItem) {
      setActiveItem(activeItem);
    }
  }, [navTree, location.pathname]);

  const handleNavItemClick = () => {
    // autoclose sidebar when layout is narrow
    if (isNarrow) {
      setNavOpen(false);
    }
  };

  return (
    <Drawer
      open={navOpen}
      variant="permanent"
      anchor="left"
      data-testid="pmm-sidebar"
    >
      <NavigationHeading
        sidebarOpen={navOpen}
        onToggleSidebar={toggleSidebar}
      />
      <List
        disablePadding
        sx={[
          { width: '100%', overflowY: 'auto', overflowX: 'hidden' },
          {
            ['.navitem-primary-text']: {
              whiteSpace: animating ? 'nowrap' : 'normal',
            },
          },
        ]}
      >
        {navTree.map((item) => (
          <SidebarNavItem
            key={item.id}
            item={item}
            activeItem={activeItem}
            drawerOpen={navOpen}
            onClick={handleNavItemClick}
          />
        ))}
      </List>
    </Drawer>
  );
};
