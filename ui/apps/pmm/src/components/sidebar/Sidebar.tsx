import { List, ListItem, ListItemButton, Stack } from '@mui/material';
import { FC } from 'react';
import { PmmRoundedIcon } from 'icons';
import { Link } from 'react-router-dom';
import { PMM_NEW_NAV_PATH } from 'lib/constants';

const items = [
  {
    name: 'Home page',
    url: PMM_NEW_NAV_PATH + '/',
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
          <ListItem disablePadding>
            {/* @ts-ignore */}
            <ListItemButton
              disableGutters
              sx={{ px: 4 }}
              LinkComponent={Link}
              to={item.url}
            >
              {item.name}
            </ListItemButton>
          </ListItem>
        ))}
      </List>
    </Stack>
  );
};
