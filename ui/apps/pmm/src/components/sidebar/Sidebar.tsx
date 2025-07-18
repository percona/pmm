import { FC, useCallback, useState } from 'react';
import { useNavigation } from 'contexts/navigation';
import { NavigationHeading } from './nav-heading';
import { Drawer } from './drawer';
import { NavItem } from './nav-item';
import List from '@mui/material/List';

export const Sidebar: FC = () => {
  const { navTree } = useNavigation();
  const [open, setIsOpen] = useState(true);

  const toggleSidebar = useCallback(() => {
    setIsOpen((prev) => !prev);
  }, []);

  return (
    <Drawer
      open={open}
      variant="permanent"
      anchor="left"
      data-testid="pmm-sidebar"
    >
      <NavigationHeading sidebarOpen={open} onToggleSidebar={toggleSidebar} />
      <List disablePadding sx={{ width: '100%', overflowY: 'auto' }}>
        {navTree.map((item) => (
          <NavItem key={item.id} item={item} drawerOpen={open} />
        ))}
      </List>
    </Drawer>
  );
};
