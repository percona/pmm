import KeyboardDoubleArrowLeftIcon from '@mui/icons-material/KeyboardDoubleArrowLeft';
import KeyboardDoubleArrowRightIcon from '@mui/icons-material/KeyboardDoubleArrowRight';
import { FC, memo } from 'react';
import { NavigationHeadingProps } from './NavigationHeading.types';
import { Icon } from 'components/icon';
import IconButton from '@mui/material/IconButton';
import Stack from '@mui/material/Stack';
import {
  appearOnOpenMixin,
  drawerTransition,
  logoMixin,
} from '../drawer/Drawer.styles';
import { DRAWER_CLOSED_WIDTH } from '../drawer/Drawer.constants';

const NavigationHeading: FC<NavigationHeadingProps> = memo(
  ({ sidebarOpen, onToggleSidebar }) => (
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
      <Stack
        sx={[
          {
            width: '150px',
            height: '48px',

            '.shown-on-hover': {
              position: 'absolute',
              top: 8,
              left: -999,
            },
          },
          !sidebarOpen && {
            width: DRAWER_CLOSED_WIDTH,

            '&:hover, &:focus-within': {
              '.hidden-on-hover': {
                visibility: 'hidden',
              },
              '.shown-on-hover': {
                left: 4,
              },
            },
          },
        ]}
      >
        {/* Collapsed, only the triangle shows: sized to keep ~9px either side
            in the rail, with its centroid on the expand button's centre. */}
        <Icon
          name="pmm-titled"
          className="hidden-on-hover"
          sx={(theme) => ({
            left: sidebarOpen ? 14 : 5,
            top: sidebarOpen ? 12 : 9,
            height: sidebarOpen ? '48px' : '38px',
            width: 'auto',
            color: sidebarOpen ? undefined : 'transparent',
            position: 'absolute',
            ...logoMixin(theme, sidebarOpen),
          })}
        />
        {!sidebarOpen && (
          <Stack
            className="shown-on-hover"
            alignItems="center"
            justifyContent="center"
          >
            <IconButton
              onClick={onToggleSidebar}
              aria-label="Expand sidebar"
              data-testid="sidebar-open-button"
            >
              <KeyboardDoubleArrowRightIcon />
            </IconButton>
          </Stack>
        )}
      </Stack>
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
  )
);

export default NavigationHeading;
