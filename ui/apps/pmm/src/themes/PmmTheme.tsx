import { listItemTextClasses } from '@mui/material/ListItemText';
import { ThemeOptions } from '@mui/material/styles';
import { deepmerge } from '@mui/utils';
import { baseThemeOptions } from '@percona/design';
import { ColorMode, PEAK_DARK_THEME, PEAK_LIGHT_THEME } from '@pmm/shared';

import '@fontsource/roboto/300.css';
import '@fontsource/roboto/400.css';
import '@fontsource/roboto/500.css';
import '@fontsource/roboto/700.css';

import '@fontsource/poppins/400.css';
import '@fontsource/poppins/500.css';
import '@fontsource/poppins/600.css';

import '@fontsource/roboto-mono';
import { iconButtonClasses } from '@mui/material/IconButton';
import { listItemIconClasses } from '@mui/material/ListItemIcon';
import { listItemButtonClasses } from '@mui/material/ListItemButton';

const perconaThemeOptions = (mode: ColorMode): ThemeOptions => {
  const peakTheme = mode === 'light' ? PEAK_LIGHT_THEME : PEAK_DARK_THEME;
  const newOptions: ThemeOptions = {
    palette: {
      mode,
      ...(mode === 'light'
        ? {
            primary: {
              main: peakTheme.primary.pmm.main,
              dark: peakTheme.primary.pmm.dark,
              light: peakTheme.primary.pmm.light,
              contrastText: peakTheme.primary.pmm.contrast,
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
              main: peakTheme.primary.pmm.main,
              dark: peakTheme.primary.pmm.dark,
              light: peakTheme.primary.pmm.light,
              contrastText: peakTheme.primary.pmm.contrast,
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
            backgroundColor: peakTheme.surfaces.elevation0,
            scrollbarColor: `${theme.palette.divider} ${theme.palette.background.paper}`,
          },
          body: {
            backgroundColor: peakTheme.surfaces.elevation0,
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
      MuiDrawer: {
        styleOverrides: {
          root: {
            fontFamily: 'Poppins',

            [`.${listItemTextClasses.root} *`]: {
              fontFamily: 'Poppins',
            },

            [`.${iconButtonClasses.root}:hover`]: {
              color: peakTheme.primary.pmm.main,
              backgroundColor: peakTheme.primary.pmm.hover,
            },

            [`.${iconButtonClasses.root}:focus`]: {
              color: peakTheme.primary.pmm.main,
              backgroundColor: peakTheme.primary.pmm.focus,
            },
          },
        },
      },
      MuiMenuItem: {
        styleOverrides: {
          root: {
            fontFamily: 'Poppins',
          },
        },
      },
      MuiListItemButton: {
        styleOverrides: {
          root: {
            '&:hover, &:focus': {
              color: peakTheme.primary.pmm.main,
              backgroundColor: peakTheme.primary.pmm.hover,

              [`.${listItemIconClasses.root}`]: {
                color: peakTheme.primary.pmm.main,
              },

              [`&.${listItemButtonClasses.selected}`]: {
                color: peakTheme.primary.pmm.main,
                backgroundColor: peakTheme.primary.pmm.selected,
              },
            },
          },
        },
      },
    },
  };

  return deepmerge<ThemeOptions>(baseThemeOptions(mode), newOptions);
};

export default perconaThemeOptions;
