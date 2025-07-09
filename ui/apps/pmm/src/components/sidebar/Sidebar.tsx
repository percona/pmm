import { List } from '@mui/material';
import { FC, useState } from 'react';
import { useNavigation } from 'contexts/navigation';
import { NavigationHeading } from './nav-heading';
import { Drawer } from './drawer';
import { NavItem } from './nav-item';

export const Sidebar: FC = () => {
  const { navTree } = useNavigation();
  const [open, setIsOpen] = useState(true);

  return (
    <Drawer
      open={open}
      variant="permanent"
      anchor="left"
      sx={(theme) => ({
        color: theme.palette.text.primary,
      })}
      data-testid="pmm-sidebar"
    >
      <NavigationHeading
        sidebarOpen={open}
        onToggleSidebar={() => setIsOpen(!open)}
      />
      <List disablePadding sx={{ width: '100%', overflowY: 'auto' }}>
        {navTree.map((item) => (
          <NavItem key={item.url} item={item} drawerOpen={open} />
        ))}
      </List>
    </Drawer>
  );
};
