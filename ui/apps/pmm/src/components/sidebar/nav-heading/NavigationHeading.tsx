import KeyboardDoubleArrowLeftIcon from '@mui/icons-material/KeyboardDoubleArrowLeft';
import KeyboardDoubleArrowRightIcon from '@mui/icons-material/KeyboardDoubleArrowRight';
import { FC, memo } from 'react';
import { Link } from 'react-router-dom';
import { PMM_NEW_NAV_USER_HOME_URL } from 'lib/constants';
import { NavigationHeadingProps } from './NavigationHeading.types';
import { Icon } from 'components/icon';
import Box from '@mui/material/Box';
import Collapse from '@mui/material/Collapse';
import IconButton from '@mui/material/IconButton';
import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
import {
  appearOnOpenMixin,
  drawerTransition,
  logoMixin,
} from '../drawer/Drawer.styles';
import {
  DRAWER_CLOSE_DURATION,
  DRAWER_CLOSED_WIDTH,
  DRAWER_OPEN_DURATION,
} from '../drawer/Drawer.constants';

const NavigationHeading: FC<NavigationHeadingProps> = memo(
  ({ sidebarOpen, onToggleSidebar, onNavigate }) => (
    <>
      <Stack
        direction="row"
        justifyContent={sidebarOpen ? 'space-between' : 'flex-start'}
        alignItems="center"
        sx={[
          (theme) => ({
            position: 'relative',
            // The header's height follows the drawer, so the list below glides
            // into place rather than jumping.
            transition: theme.transitions.create(
              'padding',
              drawerTransition(theme, sidebarOpen)
            ),
          }),
          sidebarOpen
            ? {
                pt: 1.5,
                // Lines the close button up with the group chevrons below.
                pr: 0.5,
                pb: 1.5,
                pl: 1.75,
              }
            : {
                pt: 0,
                px: 0,
                pb: 0,
              },
        ]}
      >
        <Box
          sx={{
            width: sidebarOpen ? '150px' : DRAWER_CLOSED_WIDTH,
            height: '48px',
          }}
        >
          {/* Collapsed, only the triangle shows: sized to keep ~9px either
              side in the rail. Open, the title says enough without a tooltip. */}
          <Tooltip
            title={sidebarOpen ? '' : 'Home'}
            placement="right"
          >
            <Box
              component={Link}
              to={PMM_NEW_NAV_USER_HOME_URL}
              aria-label="Home"
              data-testid="sidebar-logo-link"
              onClick={onNavigate}
              sx={(theme) => ({
                position: 'absolute',
                display: 'block',
                left: sidebarOpen ? 14 : 5,
                top: sidebarOpen ? 12 : 9,
                height: sidebarOpen ? '48px' : '38px',
                // Clips the faded title so the link, and its tooltip, end
                // at the triangle.
                maxWidth: sidebarOpen ? '150px' : '38px',
                overflow: 'hidden',
                color: 'inherit',
                textDecoration: 'none',
                borderRadius: 1,
                '&:visited, &:hover, &:active': { color: 'inherit' },
                '&:focus-visible': {
                  outline: `2px solid ${theme.palette.primary.main}`,
                },
                transition: theme.transitions.create(
                  ['left', 'top', 'height', 'max-width'],
                  drawerTransition(theme, sidebarOpen)
                ),
              })}
            >
              <Icon
                name="pmm-titled"
                sx={(theme) => ({
                  display: 'block',
                  height: '100%',
                  width: 'auto',
                  color: sidebarOpen ? undefined : 'transparent',
                  ...logoMixin(theme, sidebarOpen),
                })}
              />
            </Box>
          </Tooltip>
        </Box>
        {sidebarOpen && (
          <IconButton
            onClick={onToggleSidebar}
            aria-label="Collapse sidebar"
            data-testid="sidebar-close-button"
            sx={appearOnOpenMixin}
          >
            <KeyboardDoubleArrowLeftIcon />
          </IconButton>
        )}
      </Stack>
      {/* Collapsed, the expand button takes the first row of the rail, centred
          on the icons below. */}
      <Collapse
        in={!sidebarOpen}
        timeout={{ enter: DRAWER_CLOSE_DURATION, exit: DRAWER_OPEN_DURATION }}
        unmountOnExit
      >
        <Stack alignItems="center" sx={{ width: DRAWER_CLOSED_WIDTH }}>
          <IconButton
            onClick={onToggleSidebar}
            aria-label="Expand sidebar"
            data-testid="sidebar-open-button"
          >
            <KeyboardDoubleArrowRightIcon />
          </IconButton>
        </Stack>
      </Collapse>
    </>
  )
);

export default NavigationHeading;
