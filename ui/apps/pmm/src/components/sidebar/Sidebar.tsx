import { List, ListItem, ListItemButton, Stack } from '@mui/material';
import { FC, useEffect } from 'react';
import { PmmRoundedIcon } from 'icons';
import { Link, useLocation, useMatch } from 'react-router-dom';
import { PMM_NEW_NAV_PATH } from 'lib/constants';

const items = [
  {
    name: 'Home page',
    url: PMM_NEW_NAV_PATH + '/graph/d/pmm-home',
  },
  {
    name: 'Alerts',
    url: PMM_NEW_NAV_PATH + '/graph/alerting/alerts',
  },
  {
    name: 'Updates',
    url: PMM_NEW_NAV_PATH + '/updates',
  },
  {
    name: 'Help',
    url: PMM_NEW_NAV_PATH + '/help',
  },
];

export const NavItem: FC<{ item: (typeof items)[0] }> = ({ item }) => {
  const match = useMatch({
    path: item.url,
    end: false,
  });

  return (
    <ListItem key={item.url} disablePadding>
      {/* @ts-ignore */}
      <ListItemButton
        disableGutters
        sx={{ px: 4, fontWeight: !!match ? 'bold' : '' }}
        LinkComponent={Link}
        to={item.url}
      >
        {item.name}
      </ListItemButton>
    </ListItem>
  );
};

export const Sidebar: FC = () => {
  return (
    <Stack
      direction="column"
      sx={(theme) => ({
        gap: 1,
        backgroundColor: theme.palette.common.white,
        alignItems: 'flex-start',
      })}
    >
      <Stack
        sx={{
          pt: 2,
          px: 4,
        }}
      >
        <PmmRoundedIcon sx={{ height: 30, width: 'auto' }} />
      </Stack>
      <List disablePadding sx={{ width: '100%' }}>
        {items.map((item) => (
          <NavItem key={item.url} item={item} />
        ))}
      </List>
    </Stack>
  );
};
