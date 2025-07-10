import KeyboardDoubleArrowLeftIcon from '@mui/icons-material/KeyboardDoubleArrowLeft';
import KeyboardDoubleArrowRightIcon from '@mui/icons-material/KeyboardDoubleArrowRight';
import { IconButton } from '@mui/material';
import Stack from '@mui/material/Stack';
import { PmmRoundedIcon, PmmWithTextIcon } from 'icons';
import { FC } from 'react';
import { NavigationHeadingProps } from './NavigationHeading.types';

const NavigationHeading: FC<NavigationHeadingProps> = ({
  sidebarOpen,
  onToggleSidebar,
}) => (
  <Stack
    direction="row"
    justifyContent="space-between"
    sx={{
      p: 2,
      pl: sidebarOpen ? 2 : 3.5,
      pr: 1,
      width: '100%',
    }}
  >
    <Stack
      sx={[
        {
          width: '150px',
          height: '40px',

          '.hidden-on-hover': {
            display: 'flex',
          },
          '.shown-on-hover': {
            display: 'none',
          },
        },
        !sidebarOpen && {
          width: '40px',

          '&:hover': {
            '.hidden-on-hover': {
              display: 'none',
            },
            '.shown-on-hover': {
              display: 'flex',
            },
          },
        },
      ]}
    >
      {sidebarOpen ? (
        <PmmWithTextIcon
          className="hidden-on-hover"
          sx={{
            height: '40px',
            width: '150px',
          }}
        />
      ) : (
        <PmmRoundedIcon
          className="hidden-on-hover"
          sx={{
            height: '40px',
            width: '40px',
          }}
        />
      )}
      <IconButton
        className="shown-on-hover"
        onClick={onToggleSidebar}
        data-testid="sidebar-open-button"
      >
        <KeyboardDoubleArrowRightIcon />
      </IconButton>
    </Stack>
    {sidebarOpen && (
      <IconButton onClick={onToggleSidebar} data-testid="sidebar-close-button">
        <KeyboardDoubleArrowLeftIcon />
      </IconButton>
    )}
  </Stack>
);

export default NavigationHeading;
