import KeyboardDoubleArrowLeftIcon from '@mui/icons-material/KeyboardDoubleArrowLeft';
import KeyboardDoubleArrowRightIcon from '@mui/icons-material/KeyboardDoubleArrowRight';
import Stack from '@mui/material/Stack';
import { FC, memo } from 'react';
import { NavigationHeadingProps } from './NavigationHeading.types';
import { Icon } from 'components/icon';
import IconButton from '@mui/material/IconButton';
import { logoMixin } from '../drawer/Drawer.styles';

const NavigationHeading: FC<NavigationHeadingProps> = memo(
  ({ sidebarOpen, onToggleSidebar }) => (
    <Stack
      direction="row"
      justifyContent={sidebarOpen ? 'space-between' : 'flex-start'}
      alignItems="center"
      sx={[
        {
          position: 'relative',
        },
        sidebarOpen
          ? {
              p: 2,
            }
          : {
              px: 1,
              py: 2,
            },
      ]}
    >
      <Stack
        sx={[
          {
            width: '150px',
            height: '40px',

            '.shown-on-hover': {
              position: 'absolute',
              left: -999,
            },
          },
          !sidebarOpen && {
            width: '40px',

            '&:hover, &:focus-within': {
              '.hidden-on-hover': {
                visibility: 'hidden',
              },
              '.shown-on-hover': {
                left: 8,
              },
            },
          },
        ]}
      >
        <Icon
          name="pmm-titled"
          className="hidden-on-hover"
          sx={(theme) => ({
            left: sidebarOpen ? 16 : 8,
            height: '40px',
            width: 'auto',
            position: 'absolute',
            ...logoMixin(theme),
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
          data-testid="sidebar-close-button"
        >
          <KeyboardDoubleArrowLeftIcon />
        </IconButton>
      )}
    </Stack>
  )
);

export default NavigationHeading;
