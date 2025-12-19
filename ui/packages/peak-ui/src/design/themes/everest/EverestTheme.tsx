import { PaletteMode } from '@mui/material';
import { ThemeOptions } from '@mui/material/styles';
import { deepmerge } from '@mui/utils';
import baseThemeOptions from '../base';

const everestThemeOptions = (mode: PaletteMode): ThemeOptions => {
  const newOptions: ThemeOptions = {
    palette: {
      mode,
      ...(mode === 'light'
        ? {
            primary: {
              main: '#0E5FB5',
              dark: '#0B4A8C',
              light: '#127AE8',
              contrastText: '#FFFFFF',
            },
            action: {
              hover: 'rgba(18, 119, 227, 0.04)',
              hoverOpacity: 0.04,
              selected: 'rgba(18, 119, 227, 0.08)',
              selectedOpacity: 0.08,
              focus: 'rgba(18, 119, 227, 0.12)',
              focusOpacity: 0.12,
              focusVisible: 'rgba(18, 119, 227, 0.3)',
              focusVisibleOpacity: 0.3,
              outlinedBorder: 'rgba(18, 119, 227, 0.5)',
              outlinedBorderOpacity: 0.5,
            },
          }
        : {
            primary: {
              main: '#62AEFF',
              dark: '#1486FF',
              light: '#B6D9FF',
              contrastText: '#000000',
            },
            action: {
              hover: 'rgba(46, 147, 255, 0.08)',
              hoverOpacity: 0.08,
              selected: 'rgba(46, 147, 255, 0.12)',
              selectedOpacity: 0.12,
              focus: 'rgba(46, 147, 255, 0.15)',
              focusOpacity: 0.15,
              focusVisible: 'rgba(46, 147, 255, 0.3)',
              focusVisibleOpacity: 0.3,
              outlinedBorder: 'rgba(46, 147, 255, 0.5)',
              outlinedBorderOpacity: 0.5,
            },
          }),
    },
    components: {
      MuiCssBaseline: {
        styleOverrides: (theme) => ({
          a: {
            color: 'inherit',
            ...theme.typography.body2,
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
          root: ({ theme }) => ({
            backgroundColor: theme.palette.background.default,
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
      MuiAccordion: {
        styleOverrides: {
          root: () => ({
            backgroundColor: 'transparent',
          }),
        },
      },
    },
  };

  return deepmerge<ThemeOptions>(baseThemeOptions(mode), newOptions);
};

export default everestThemeOptions;
