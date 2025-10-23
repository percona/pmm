import { FC, useCallback, useState } from 'react';
import { useNavigation } from 'contexts/navigation';
import { NavigationHeading } from './nav-heading';
import { Drawer } from './drawer';
import { NavItem } from './nav-item';
import List from '@mui/material/List';

export const Sidebar: FC = () => {
  const { navTree, navOpen, setNavOpen } = useNavigation();

  const [animating, setAnimating] = useState(false);

  const toggleSidebar = useCallback(() => {
    setNavOpen(!navOpen);

    setAnimating(true);

    const timeoutId = setTimeout(() => setAnimating(false), 2000);

    return () => {
      clearTimeout(timeoutId);
    };
  }, [navOpen, setNavOpen]);

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
          <NavItem key={item.id} item={item} drawerOpen={navOpen} />
        ))}
      </List>
    </Drawer>
  );
};
