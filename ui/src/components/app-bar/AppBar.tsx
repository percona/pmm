import {
  Box,
  AppBar as MuiAppBar,
  Toolbar,
  Typography,
  Link,
  Stack,
} from '@mui/material';
import { HelpFilledIcon, PmmRoundedIcon } from 'icons';
import { Breadcrumbs } from 'components/breadcrumbs';
import { PMM_SUPPORT_URL } from 'constants';
import { Messages } from './AppBar.messages';
import { HomeLink } from 'components/home-link';

export const AppBar = () => (
  <MuiAppBar position="sticky" color="primary">
    <Toolbar>
      <HomeLink
        color="inherit"
        underline="hover"
        sx={{
          mr: 2,
        }}
        data-testid="appbar-pmm-link"
      >
        <Stack gap={1} direction="row" alignItems="center">
          <PmmRoundedIcon sx={{ height: '40px', width: 'auto' }} />
          <Typography>{Messages.title}</Typography>
        </Stack>
      </HomeLink>
      <Breadcrumbs />
      <Box sx={{ ml: 'auto' }}>
        <Link
          href={PMM_SUPPORT_URL}
          target="_blank"
          rel="noopener noreferrer"
          color="inherit"
          underline="hover"
          data-testid="appbar-support-link"
        >
          <Stack gap={1} direction="row" alignItems="center">
            <HelpFilledIcon />
            <Typography>{Messages.support}</Typography>
          </Stack>
        </Link>
      </Box>
    </Toolbar>
  </MuiAppBar>
);
