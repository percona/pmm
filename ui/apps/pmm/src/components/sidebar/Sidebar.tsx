import { FC, useCallback, useState } from 'react';
import { useNavigation } from 'contexts/navigation';
import { NavigationHeading } from './nav-heading';
import { Drawer } from './drawer';
import { NavItem } from './nav-item';
import List from '@mui/material/List';
import { useTheme } from '@mui/material/styles';

export const Sidebar: FC = () => {
  const theme = useTheme();
  const { navTree } = useNavigation();
  const [open, setIsOpen] = useState(true);
  const [animating, setAnimating] = useState(false);

  const toggleSidebar = useCallback(() => {
    const { enteringScreen, leavingScreen } = theme.transitions.duration;

    setIsOpen((prev) => !prev);

    setAnimating(true);
    setTimeout(
      () => setAnimating(false),
      open ? leavingScreen : enteringScreen
    );
  }, [open, theme.transitions.duration]);

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
