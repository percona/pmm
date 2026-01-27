import { listItemTextClasses } from "@mui/material/ListItemText";
import { PaletteMode, ThemeOptions } from "@mui/material";
import { deepmerge } from "@mui/utils";

import { iconButtonClasses } from "@mui/material/IconButton";
import { listItemIconClasses } from "@mui/material/ListItemIcon";
import { listItemButtonClasses } from "@mui/material/ListItemButton";
import baseThemeOptions, { 
  primitives,
  semanticTokensLight,
  semanticTokensDark,
} from "../base";

// PMM Primary color tokens
const pmmPrimaryLight = {
  main: primitives.brand.fire[600],
  dark: primitives.brand.fire[700],
  light: primitives.brand.fire[500],
  contrastText: primitives.base.white,
  hover: "rgba(242, 69, 0, 0.04)",
  selected: "rgba(242, 69, 0, 0.08)",
  focus: "rgba(242, 69, 0, 0.12)",
  focusVisible: "rgba(242, 69, 0, 0.3)",
  outlinedBorder: "rgba(242, 69, 0, 0.5)",
};
const pmmPrimaryDark = {
  main: primitives.brand.fire[200],
  dark: primitives.brand.fire[300],
  light: primitives.brand.fire[100],
  contrastText: primitives.base.black,
  hover: "rgba(251, 197, 176, 0.08)",
  selected: "rgba(251, 197, 176, 0.12)",
  focus: "rgba(251, 197, 176, 0.15)",
  focusVisible: "rgba(251, 197, 176, 0.3)",
  outlinedBorder: "rgba(251, 197, 176, 0.5)",
};

// PMM semantic color tokens
const pmmTokensLight = { ...semanticTokensLight };
const pmmTokensDark = { ...semanticTokensDark };

const pmmThemeOptions = (mode: PaletteMode): ThemeOptions => {
  const tokens = mode === "light" ? pmmTokensLight : pmmTokensDark;
  const primary = mode === "light" ? pmmPrimaryLight : pmmPrimaryDark;
  
  const newOptions: ThemeOptions = {
    palette: {
      mode,
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
            ...(ownerState.color === "primary" && {
              color: theme.palette.primary.main,
            }),
            "&:hover": {
              backgroundColor: theme.palette.action.selected,
            },
            "&:focus": {
              backgroundColor: theme.palette.action.focusVisible,
            },
            ...(ownerState.size === "large" && {
              svg: {
                width: 40,
                height: 40,
              },
            }),
            ...(ownerState.size === "small" && {
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
            color: "#FBFBFB",
            backgroundColor: "#3A4151",
          }),
        },
      },
      MuiToggleButtonGroup: {
        styleOverrides: {
          root: ({ theme }) => ({
            [theme.breakpoints.down("sm")]: {
              flexDirection: "column",
            },
          }),
        },
      },
      MuiLinearProgress: {
        styleOverrides: {
          root: ({ theme }) => ({
            height: 10,
            borderStyle: "solid",
            borderRadius: 5,
            borderColor: theme.palette.divider,
            backgroundColor: theme.palette.surfaces?.low,
          }),
          bar: {
            borderRadius: 5,
            backgroundColor: "#606C86",
          },
        },
      },
      MuiChip: {
        variants: [
          {
            props: { variant: "filled", color: "warning" },
            style: {
              color: tokens.warning.contrastText,
            },
          },
        ],
        styleOverrides: {
          icon: {
            width: 22,
            height: 22,
          },
          colorError: {
            color: "#920000",
            backgroundColor: "#FFECE9",
          },
          colorWarning: {
            color: tokens.warning.light,
            borderColor: tokens.warning.main,
          },
        },
      },
      MuiBadge: {
        styleOverrides: {
          colorWarning: {
            backgroundColor: tokens.warning.light,
          },
        },
      },
      MuiCard: {
        defaultProps: {
          variant: "outlined",
        },
      },
      MuiDrawer: {
        styleOverrides: {
          root: {
            fontFamily: "Poppins",

            [`.${listItemTextClasses.root} *`]: {
              fontFamily: "Poppins",
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
            fontFamily: "Poppins",
          },
        },
      },
      MuiListItemButton: {
        styleOverrides: {
          root: {
            "&:hover, &:focus": {
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
            color: tokens.neutral.contrastText,
            backgroundColor: tokens.neutral.main,
          }),
          arrow: () => ({
            color: tokens.neutral.main,
          }),
        },
      },
    },
  };

  return deepmerge<ThemeOptions>(baseThemeOptions(mode), newOptions);
};

export default pmmThemeOptions;
