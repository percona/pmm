import { PaletteMode } from '@mui/material';
import { ThemeOptions } from '@mui/material/styles';
import { deepmerge } from '@mui/utils';
import { baseThemeOptions } from '@percona/design';

import '@fontsource/roboto/300.css';
import '@fontsource/roboto/400.css';
import '@fontsource/roboto/500.css';
import '@fontsource/roboto/700.css';

import '@fontsource/poppins/400.css';
import '@fontsource/poppins/500.css';
import '@fontsource/poppins/600.css';

import '@fontsource/roboto-mono';

const perconaThemeOptions = (mode: PaletteMode): ThemeOptions => {
  const newOptions: ThemeOptions = {
    palette: {
      mode,
      ...(mode === 'light'
        ? {
            primary: {
              main: '#AC3100',
              dark: '#852600',
              light: '#DC3F00',
              contrastText: '#FFFFFF',
            },
            action: {
              hover: 'rgba(220, 63, 0, 0.04)',
              hoverOpacity: 0.04,
              selected: 'rgba(220, 63, 0, 0.08)',
              selectedOpacity: 0.08,
              focus: 'rgba(220, 63, 0, 0.12)',
              focusOpacity: 0.12,
              focusVisible: 'rgba(220, 48, 0, 0.3)',
              focusVisibleOpacity: 0.3,
              outlinedBorder: 'rgba(220, 33, 0, 0.5)',
              outlinedBorderOpacity: 0.5,
            },
          }
        : {
            primary: {
              main: '#F68254',
              dark: '#F56A33',
              light: '#F9A98A',
              contrastText: '#000000',
            },
            action: {
              hover: 'rgba(245, 106, 51, 0.08)',
              hoverOpacity: 0.08,
              selected: 'rgba(245, 106, 51, 0.12)',
              selectedOpacity: 0.12,
              focus: 'rgba(245, 106, 51, 0.15)',
              focusOpacity: 0.15,
              focusVisible: 'rgba(245, 106, 51, 0.3)',
              focusVisibleOpacity: 0.3,
              outlinedBorder: 'rgba(245, 106, 51, 0.5)',
              outlinedBorderOpacity: 0.5,
            },
          }),
    },
    components: {
      MuiCssBaseline: {
        styleOverrides: (theme) => ({
          html: {
            backgroundColor: mode === 'light' ? undefined : '#3A4151',
            scrollbarColor: `${theme.palette.divider} ${theme.palette.background.paper}`,
          },
          body: {
            backgroundColor: mode === 'light' ? undefined : '#3A4151',

            scrollbarColor: `${theme.palette.divider} ${theme.palette.background.paper}`,
          },
        }),
      },
      MuiIconButton: {
        defaultProps: {
          disableTouchRipple: true,
        },
        styleOverrides: {
          root: ({ theme, ownerState }) => ({
            color: theme.palette.text.primary,
            ...(ownerState.color === 'primary' && {
              color: theme.palette.primary.main,
            }),
            '&:hover': {
              backgroundColor: theme.palette.action.selected,
            },
            '&:focus': {
              backgroundColor: theme.palette.action.focusVisible,
            },
            ...(ownerState.size === 'large' && {
              svg: {
                width: 40,
                height: 40,
              },
            }),
            ...(ownerState.size === 'small' && {
              svg: {
                width: 20,
                height: 20,
              },
            }),
          }),
        },
      },
      MuiAppBar: {
        styleOverrides: {
          root: () => ({
            color: '#FBFBFB',
            backgroundColor: '#3A4151',
          }),
        },
      },
      MuiToggleButtonGroup: {
        styleOverrides: {
          root: ({ theme }) => ({
            [theme.breakpoints.down('sm')]: {
              flexDirection: 'column',
            },
          }),
        },
      },
      MuiLinearProgress: {
        styleOverrides: {
          root: ({ theme }) => ({
            height: 10,
            borderStyle: 'solid',
            borderRadius: 5,
            borderColor: theme.palette.divider,
            backgroundColor: theme.palette.surfaces?.low,
          }),
          bar: {
            borderRadius: 5,
            backgroundColor: '#606C86',
          },
        },
      },
      MuiChip: {
        styleOverrides: {
          icon: {
            width: 22,
            height: 22,
          },
          colorError: {
            color: '#920000',
            backgroundColor: '#FFECE9',
          },
        },
      },
      MuiCard: {
        defaultProps: {
          variant: 'outlined',
        },
      },
      MuiButton: {
        styleOverrides: {
          textSizeMedium: {
            padding: '11px 16px',
            height: '42px',
          },
        },
      },
    },
  };

  return deepmerge<ThemeOptions>(baseThemeOptions(mode), newOptions);
};

export default perconaThemeOptions;
