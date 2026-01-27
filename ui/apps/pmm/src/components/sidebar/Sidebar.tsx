import { FC, useCallback, useEffect, useState } from 'react';
import { useNavigation } from 'contexts/navigation';
import { NavigationHeading } from './nav-heading';
import { Drawer } from './drawer';
import { NavItem } from './nav-item';
import List from '@mui/material/List';
import { findActiveNavItem } from 'utils/navigation.utils';
import { useLocation } from 'react-router-dom';
import { NavItem as NavItemType } from 'types/navigation.types';

export const Sidebar: FC = () => {
  const { navTree, navOpen, setNavOpen } = useNavigation();
  const [activeItem, setActiveItem] = useState<NavItemType>(navTree[0]);
  const [animating, setAnimating] = useState(false);
  const location = useLocation();

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
          <NavItem
            key={item.id}
            item={item}
            activeItem={activeItem}
            drawerOpen={navOpen}
          />
        ))}
      </List>
    </Drawer>
  );
};
