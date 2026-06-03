import KeyboardDoubleArrowLeftIcon from '@mui/icons-material/KeyboardDoubleArrowLeft';
import KeyboardDoubleArrowRightIcon from '@mui/icons-material/KeyboardDoubleArrowRight';
import { FC, memo } from 'react';
import { NavigationHeadingProps } from './NavigationHeading.types';
import { Icon } from 'components/icon';
import IconButton from '@mui/material/IconButton';
import Stack from '@mui/material/Stack';
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
              pt: 1.5,
              pr: 0.875,
              pb: 0.5,
              pl: 1.75,
            }
          : {
              pt: 1.5,
              px: 1,
              pb: 0.5,
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
              top: 16,
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
            left: sidebarOpen ? 14 : 8,
            top: sidebarOpen ? 12 : 16,
            height: sidebarOpen ? '48px' : '40px',
            width: 'auto',
            color: sidebarOpen ? undefined : 'transparent',
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
