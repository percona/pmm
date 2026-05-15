import { PaletteMode, ThemeOptions } from '@mui/material';
import { deepmerge } from '@mui/utils';
import { iconButtonClasses } from '@mui/material/IconButton';
import { listItemIconClasses } from '@mui/material/ListItemIcon';
import { listItemButtonClasses } from '@mui/material/ListItemButton';

import baseThemeOptions, { semanticTokensLight, semanticTokensDark } from '../base';

// SEP brand colors
const sepBrand = {
  purple: {
    900: '#2A1A66',
    800: '#3B2790',
    700: '#4D32C3',
    600: '#5A37E0',
    500: '#653DF4',
    400: '#8464F6',
    300: '#9D83F8',
    200: '#BEB0FA',
    100: '#D9D0FC',
    50: '#F2EFFE',
  },
  yellow: {
    900: '#656A23',
    800: '#8A9020',
    700: '#B3BA1C',
    600: '#DCE518',
    500: '#F6FE54',
    400: '#F8FE76',
    300: '#F9FE8E',
    200: '#FBFEB5',
    100: '#FCFED2',
    50: '#FEFFF0',
  },
  black: '#282727',
  white: '#FFFFFF',
} as const;

// Technology colors for charts/badges
export const sepTechnologyColors = {
  mysql: '#E65A15',
  postgresql: '#005ED6',
  mongodb: '#1FA23A',
  kubernetes: '#2AA6DF',
  redis: '#D6362A',
  valkey: '#A83FEF',
} as const;

// SEP Primary color tokens
export const sepPrimaryLight = {
  main: sepBrand.purple[500],
  dark: sepBrand.purple[700],
  light: sepBrand.purple[400],
  contrastText: sepBrand.white,
  hover: 'rgba(101, 61, 244, 0.04)',
  selected: 'rgba(101, 61, 244, 0.08)',
  focus: 'rgba(101, 61, 244, 0.12)',
  focusVisible: 'rgba(101, 61, 244, 0.3)',
  outlinedBorder: 'rgba(101, 61, 244, 0.5)',
};

export const sepPrimaryDark = {
  main: sepBrand.purple[200],
  dark: sepBrand.purple[300],
  light: sepBrand.purple[100],
  contrastText: sepBrand.black,
  hover: 'rgba(190, 176, 250, 0.08)',
  selected: 'rgba(190, 176, 250, 0.12)',
  focus: 'rgba(190, 176, 250, 0.15)',
  focusVisible: 'rgba(190, 176, 250, 0.3)',
  outlinedBorder: 'rgba(190, 176, 250, 0.5)',
};

// SEP semantic color tokens — light overrides
export const sepTokensLight = {
  ...semanticTokensLight,
  text: {
    ...semanticTokensLight.text,
    accent1: sepBrand.purple[600],
    accent2: sepBrand.purple[600],
  },
  charts: {
    ...semanticTokensLight.charts,
    // Add technology colors for chart use
    mysql: sepTechnologyColors.mysql,
    postgresql: sepTechnologyColors.postgresql,
    mongodb: sepTechnologyColors.mongodb,
    kubernetes: sepTechnologyColors.kubernetes,
    redis: sepTechnologyColors.redis,
    valkey: sepTechnologyColors.valkey,
  },
};

// SEP semantic color tokens — dark overrides
export const sepTokensDark = {
  ...semanticTokensDark,
  text: {
    ...semanticTokensDark.text,
    accent1: sepBrand.purple[200],
    accent2: sepBrand.purple[200],
  },
  charts: {
    ...semanticTokensDark.charts,
    mysql: sepTechnologyColors.mysql,
    postgresql: sepTechnologyColors.postgresql,
    mongodb: sepTechnologyColors.mongodb,
    kubernetes: sepTechnologyColors.kubernetes,
    redis: sepTechnologyColors.redis,
    valkey: sepTechnologyColors.valkey,
  },
};

const sepThemeOptions = (mode: PaletteMode): ThemeOptions => {
  const tokens = mode === 'light' ? sepTokensLight : sepTokensDark;
  const primary = mode === 'light' ? sepPrimaryLight : sepPrimaryDark;

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
      MuiButton: {
        variants: [
          { props: { variant: 'contained', color: 'success' }, style: { color: sepBrand.white } },
          { props: { variant: 'contained', color: 'error' }, style: { color: sepBrand.white } },
          {
            props: { variant: 'contained', color: 'warning' },
            style: { color: mode === 'light' ? sepBrand.white : 'initial' },
          },
          { props: { variant: 'contained', color: 'info' }, style: { color: sepBrand.white } },
        ],
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
            color: sepBrand.white,
            backgroundColor: primary.main,
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
            backgroundColor: tokens.surfaces.elevation0,
          }),
          bar: {
            borderRadius: 5,
            backgroundColor: sepBrand.purple[500],
          },
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
      MuiLink: {
        styleOverrides: {
          root: {
            color: mode === 'light' ? sepBrand.purple[600] : sepBrand.purple[200],
            textDecorationColor: mode === 'light' ? sepBrand.purple[600] : sepBrand.purple[200],
            '&:hover': {
              opacity: 0.8,
            },
            '&:active': {
              opacity: 0.6,
            },
            '&:focus-visible': {
              outline: `2px solid ${mode === 'light' ? sepBrand.purple[600] : sepBrand.purple[200]}`,
              outlineOffset: '2px',
              borderRadius: '2px',
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
            color: primary.contrastText,
            backgroundColor: tokens.neutral.main,
          }),
          arrow: () => ({
            color: tokens.neutral.main,
          }),
        },
      },
      MuiPaper: {
        styleOverrides: {
          root: ({ theme }) => ({
            ...(theme.palette.mode === 'light' && {
              // MuiTableSep is a custom class added to the material-react-table on SEP to apply SEP table's Toolbar and TableRow children
              '&.MuiTableSep .MuiToolbar-root': {
                backgroundColor: theme.palette.background.paper,
              },
              '&.MuiTableSep .MuiTableRow-root': {
                backgroundColor: theme.palette.background.paper + ' !important',
              },
            }),
          }),
        },
      },
      MuiInputBase: {
        styleOverrides: {
          root: ({ theme }) => ({
            ...(theme.palette.mode === 'light' && {
              backgroundColor: theme.palette.background.paper,
            }),
          }),
        },
      },
      MuiChip: {
        styleOverrides: {
          root: ({ theme, ownerState }) => ({
            ...(theme.palette.mode === 'light' &&
              ownerState.variant === 'filled' &&
              ownerState.color === 'default' && {
                // Default filled chip blends into #F6F5F5 page bg — add visible surface
                backgroundColor: 'rgba(40, 39, 39, 0.12)',
              }),
          }),
        },
      },
    },
  };

  return deepmerge<ThemeOptions>(baseThemeOptions(mode), newOptions);
};

export default sepThemeOptions;
