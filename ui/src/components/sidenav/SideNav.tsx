import {
  Notifications,
  Settings,
  ExpandLess,
  ExpandMore,
} from '@mui/icons-material';
import {
  SvgIcon,
  ListItem,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Collapse,
  Drawer,
  Toolbar,
  Stack,
  List,
} from '@mui/material';
import { useMessageWithResult } from 'contexts/messages/messages.hooks';
import { MenuItem } from 'contexts/navigation/navigation.context.types';
import { useNavigation } from 'contexts/navigation/navigation.hooks';
import { FC, useEffect, useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';

const MenuIcon: FC<{ name: string }> = ({ name }) => {
  if (name === 'dashboards') {
    return (
      <SvgIcon>
        <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
          <path
            d="M9 3V21C9 21.5304 8.78929 22.0391 8.41421 22.4142C8.03914 22.7893 7.53043 23 7 23H3C2.46957 23 1.96086 22.7893 1.58579 22.4142C1.21071 22.0391 1 21.5304 1 21V3C1 2.46957 1.21071 1.96086 1.58579 1.58579C1.96086 1.21071 2.46957 1 3 1H7C7.53043 1 8.03914 1.21071 8.41421 1.58579C8.78929 1.96086 9 2.46957 9 3ZM15 16.7H21C21.5304 16.7 22.0391 16.9107 22.4142 17.2858C22.7893 17.6609 23 18.1696 23 18.7V21C23 21.5304 22.7893 22.0391 22.4142 22.4142C22.0391 22.7893 21.5304 23 21 23H15C14.4696 23 13.9609 22.7893 13.5858 22.4142C13.2107 22.0391 13 21.5304 13 21V18.7C13 18.1696 13.2107 17.6609 13.5858 17.2858C13.9609 16.9107 14.4696 16.7 15 16.7ZM15 1H21C21.5304 1 22.0391 1.21071 22.4142 1.58579C22.7893 1.96086 23 2.46957 23 3V10.7C23 11.2304 22.7893 11.7391 22.4142 12.1142C22.0391 12.4893 21.5304 12.7 21 12.7H15C14.4696 12.7 13.9609 12.4893 13.5858 12.1142C13.2107 11.7391 13 11.2304 13 10.7V3C13 2.46957 13.2107 1.96086 13.5858 1.58579C13.9609 1.21071 14.4696 1 15 1Z"
            stroke="currentColor"
            strokeWidth="2"
          />
        </svg>
      </SvgIcon>
    );
  }
  if (name === 'alerts') {
    return <Notifications />;
  }

  if (name === 'settings') {
    return <Settings />;
  }

  return null;
};

const NavItem: FC<{ item: MenuItem }> = ({ item }) => {
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);
  const location = useLocation();
  const isActive = location.pathname.includes(item.to!);
  const [to, setTo] = useState(item.to);
  const { result, sendMessage } = useMessageWithResult();

  useEffect(() => {
    // check if dashboard
    if (location.pathname.includes('/d/')) {
      sendMessage({
        type: 'LINK_VARIABLES',
        data: {
          url: item.to,
        },
      });
    } else {
      setTo(item.to);
    }
  }, [location, item.to]);

  useEffect(() => {
    console.log(result);
    if (result && result.data?.url) {
      setTo(result.data.url);
    }
  }, [result]);

  if (!item.children) {
    return (
      <ListItem disablePadding onClick={to ? () => navigate(to) : undefined}>
        <ListItemButton>
          <ListItemIcon
            sx={{
              ml: -0.5,
              minWidth: 35,
            }}
          >
            {!!item.icon && <MenuIcon name={item.icon} />}
          </ListItemIcon>
          <ListItemText
            primary={item.title}
            sx={{
              '.MuiListItemText-primary ': {
                color: isActive ? 'primary.main' : undefined,
                fontWeight: isActive ? 'bold' : undefined,
              },
            }}
          />
        </ListItemButton>
      </ListItem>
    );
  }

  return (
    <>
      <ListItem disablePadding>
        <ListItemButton onClick={() => setOpen(!open)}>
          <ListItemIcon
            sx={{
              ml: -0.5,
              minWidth: 35,
            }}
          >
            {!!item.icon && <MenuIcon name={item.icon} />}
          </ListItemIcon>
          <ListItemText primary={item.title} />
          {open ? <ExpandLess /> : <ExpandMore />}
        </ListItemButton>
      </ListItem>
      <Collapse in={open} timeout="auto" unmountOnExit>
        {item.children?.map((child) => (
          <NavItem key={child.title} item={child} />
        ))}
      </Collapse>
    </>
  );
};

export const SideNav: FC = () => {
  const { navTree } = useNavigation();

  return (
    <Drawer
      open
      variant="permanent"
      sx={{
        width: 280,
        flexShrink: 0,
        [`& .MuiDrawer-paper`]: {
          width: 280,
          boxSizing: 'border-box',
        },
      }}
    >
      <Toolbar />
      <Stack>
        <List>
          {navTree.map((item) => (
            <NavItem key={item.title} item={item} />
          ))}
        </List>
      </Stack>
    </Drawer>
  );
};
