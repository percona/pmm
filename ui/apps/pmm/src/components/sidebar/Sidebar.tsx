import { FC, useCallback, useState } from 'react';
import { useNavigation } from 'contexts/navigation';
import { NavigationHeading } from './nav-heading';
import { Drawer } from './drawer';
import { NavItem } from './nav-item';
import List from '@mui/material/List';
import { useLocalStorage } from 'hooks/utils/useLocalStorage';

export const Sidebar: FC = () => {
  const { navTree } = useNavigation();
  const [open, setIsOpen] = useLocalStorage<boolean>(
    'pmm-ui.sidebar.expanded',
    true
  );
  const [animating, setAnimating] = useState(false);

  const toggleSidebar = useCallback(() => {
    setIsOpen(!open);

    setAnimating(true);

    const timeoutId = setTimeout(() => setAnimating(false), 2000);

    return () => {
      clearTimeout(timeoutId);
    };
  }, [open]);

  return (
    <Drawer
      open={open}
      variant="permanent"
      anchor="left"
      data-testid="pmm-sidebar"
    >
      <NavigationHeading sidebarOpen={open} onToggleSidebar={toggleSidebar} />
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
          <NavItem key={item.id} item={item} drawerOpen={open} />
        ))}
      </List>
    </Drawer>
  );
};
