import { listItemTextClasses } from '@mui/material/ListItemText';
import { PaletteMode, ThemeOptions } from '@mui/material';
import { deepmerge } from '@mui/utils';

import { iconButtonClasses } from '@mui/material/IconButton';
import { listItemIconClasses } from '@mui/material/ListItemIcon';
import { listItemButtonClasses } from '@mui/material/ListItemButton';
import baseThemeOptions, { primitives, semanticTokensLight, semanticTokensDark } from '../base';

// PMM Primary color tokens
export const pmmPrimaryLight = {
  dark: primitives.primary.purple[800],
  main: primitives.primary.purple[600],
  light: primitives.primary.purple[400],
  contrastText: primitives.pure.white,
  hover: "rgba(101, 61, 244, 0.04)",
  selected: "rgba(101, 61, 244, 0.08)",
  focus: "rgba(101, 61, 244, 0.12)",
  focusVisible: "rgba(101, 61, 244, 0.3)",
  outlinedBorder: "rgba(101, 61, 244, 0.5)",
};
export const pmmPrimaryDark = {
  dark: primitives.primary.purple[400],
  main: primitives.primary.purple[300],
  light: primitives.primary.purple[200],
  contrastText: primitives.pure.black,
  hover: "rgba(182, 178, 255, 0.08)",
  selected: "rgba(182, 178, 255, 0.20)",
  focus: "rgba(182, 178, 255, 0.30)",
  focusVisible: "rgba(182, 178, 255, 0.5)",
  outlinedBorder: "rgba(182, 178, 255, 0.75)",
};

// PMM semantic color tokens
const pmmTokensLight = { ...semanticTokensLight };
const pmmTokensDark = { ...semanticTokensDark };

const pmmThemeOptions = (mode: PaletteMode): ThemeOptions => {
  const tokens = mode === 'light' ? pmmTokensLight : pmmTokensDark;
  const primary = mode === 'light' ? pmmPrimaryLight : pmmPrimaryDark;

  const newOptions: ThemeOptions = {
    palette: {
      mode,
      surfaces: tokens.surfaces,
      primary,
    },
    components: {
      MuiCssBaseline: {
        styleOverrides: (theme) => ({
          html: {
            backgroundColor: theme.palette.background.default,
            scrollbarColor: `${theme.palette.divider} ${theme.palette.background.paper}`,
          },
          body: {
            backgroundColor: theme.palette.background.default,
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
            color: primitives.primary.black[50],
            backgroundColor: primitives.primary.black[800],
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
            backgroundColor: theme.palette.background.default,
          }),
          bar: ({ theme }) => ({
            borderRadius: 5,
            backgroundColor: theme.palette.primary.main,
          }),
        },
      },
      MuiBadge: {
        styleOverrides: {
          colorWarning: ({ theme }) => ({
            backgroundColor: theme.palette.warning.light,
            color:
              theme.palette.mode === "light"
                ? theme.palette.common.white
                : theme.palette.warning.contrastText,
          }),
        },
      },
      MuiCard: {
        defaultProps: {
          variant: 'outlined',
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
              color: primary.main,
              backgroundColor: primary.hover,
            },

            [`.${iconButtonClasses.root}:focus`]: {
              color: primary.main,
              backgroundColor: primary.focus,
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
              color: primary.main,
              backgroundColor: primary.hover,

              [`.${listItemIconClasses.root}`]: {
                color: primary.main,
              },

              [`&.${listItemButtonClasses.selected}`]: {
                color: primary.main,
                backgroundColor: primary.selected,
              },
            },
          },
        },
      },
      MuiTooltip: {
        styleOverrides: {
          tooltip: ({ theme }) => ({
            ...theme.typography.helperText,
            p: 6,
            boxShadow: theme.shadows[8],
            color: theme.palette.tooltips?.color,
            backgroundColor: theme.palette.tooltips?.background,
          }),
          arrow: ({ theme }) => ({
            color: theme.palette.tooltips?.background,
          }),
        },
      },
      MuiDialogTitle: {
        styleOverrides: {
          root: {
            paddingBottom: 0,
          },
        },
      },
      MuiDialogActions: {
        styleOverrides: {
          root: {
            paddingTop: 0,
          },
        },
      },
    },
  };

  return deepmerge<ThemeOptions>(baseThemeOptions(mode), newOptions);
};

export default pmmThemeOptions;
