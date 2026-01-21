import { listItemTextClasses } from "@mui/material/ListItemText";
import { PaletteMode, ThemeOptions } from "@mui/material";
import { deepmerge } from "@mui/utils";

import { iconButtonClasses } from "@mui/material/IconButton";
import { listItemIconClasses } from "@mui/material/ListItemIcon";
import { listItemButtonClasses } from "@mui/material/ListItemButton";
import baseThemeOptions, { primitives } from "../base";

const basisLight = {
  surfaces: {
    elevation0: primitives.brand.stone[50],
    elevation1: primitives.base.white,
    backdrop: "#2C323EB8",
  },
  text: {
    primary: "#2C323EFF",
    secondary: "#2C323EB8",
    disabled: "#2C323E66",
    sky: primitives.brand.sky[600],
    lavender: primitives.brand.lavender[600],
    aqua: primitives.brand.aqua[700],
  },
  primary: {
    main: primitives.brand.fire[600],
    dark: primitives.brand.fire[700],
    light: primitives.brand.fire[500],
    contrast: primitives.base.white,
    hover: "#F245000A",
    selected: "#F2450014",
    focus: "#F245001F",
    focusVisible: "#F245004D",
    outlinedBorder: "#F2450080",
  },
  lines: {
    divider: "#2C323E40",
    dividerStrong: "#2C323E80",
    contour: "#0000000F",
    dividerStronger: "#2C323EFF",
  },
  action: {
    hover: "#2C323E0A",
    disabled: "#2C323E1F",
    focus: "#2C323E1F",
  },
  error: {
    main: primitives.extra.red[600],
    light: primitives.extra.red[500],
    dark: primitives.extra.red[700],
    contrast: primitives.extra.red[700],
    surface: primitives.extra.red[50],
  },
  success: {
    main: primitives.extra.green[600],
    light: primitives.extra.green[500],
    dark: primitives.extra.green[700],
    contrast: primitives.extra.green[700],
    surface: primitives.extra.green[50],
  },
  informative: {
    main: primitives.brand.sky[700],
    light: primitives.brand.sky[600],
    dark: primitives.brand.sky[800],
    contrast: primitives.brand.sky[800],
    surface: primitives.brand.sky[50],
  },
  warning: {
    main: primitives.extra.yellow[800],
    light: primitives.brand.sunrise[700],
    dark: primitives.brand.sunrise[900],
    contrast: primitives.brand.sunrise[900],
    surface: primitives.extra.yellow[100],
  },
  charts: {
    stone: primitives.brand.stone[400],
    lavender: primitives.brand.lavender[500],
    sky: primitives.brand.sky[500],
    aqua: primitives.brand.aqua[600],
    sunrise: primitives.brand.sunrise[500],
    sunset: primitives.brand.sunset[400],
    red: primitives.extra.red[400],
  },
  neutral: {
    main: primitives.brand.stone[800],
    light: primitives.brand.stone[700],
    dark: primitives.brand.stone[900],
    contrast: primitives.brand.stone[800],
    surface: "#2C323E1F",
  },
};

const basisDark = {
  surfaces: {
    elevation0: primitives.brand.stone[800],
    elevation1: primitives.brand.stone[900],
    backdrop: "#525760DB",
  },
  text: {
    primary: "#FBFBFBFF",
    secondary: "#FBFBFBB8",
    disabled: "#FBFBFB66",
    sky: primitives.brand.sky[200],
    lavender: primitives.brand.lavender[200],
    aqua: primitives.brand.aqua[300],
  },
  primary: {
    main: primitives.brand.fire[200],
    dark: primitives.brand.fire[300],
    light: primitives.brand.fire[100],
    contrast: primitives.base.black,
    hover: "#FBC5B014",
    selected: "#FBC5B01F",
    focus: "#FBC5B026",
    focusVisible: "#FBC5B04D",
    outlinedBorder: "#FBC5B080",
  },
  lines: {
    divider: "#FFFFFF40",
    dividerStrong: "#FFFFFF80",
    contour: "#FFFFFF14",
    dividerStronger: "#FFFFFFFF",
  },
  action: {
    hover: "#F0F1F414",
    disabled: "#F0F1F426",
    focus: "#F0F1F426",
  },
  error: {
    main: primitives.extra.red[300],
    light: primitives.extra.red[200],
    dark: primitives.extra.red[400],
    contrast: primitives.base.white,
    surface: primitives.extra.red[400],
  },
  success: {
    main: primitives.extra.green[400],
    light: primitives.extra.green[300],
    dark: primitives.extra.green[500],
    contrast: primitives.base.white,
    surface: primitives.extra.green[500],
  },
  informative: {
    main: primitives.brand.sky[500],
    light: primitives.brand.sky[400],
    dark: primitives.brand.sky[600],
    contrast: primitives.base.white,
    surface: primitives.brand.sky[600],
  },
  warning: {
    main: primitives.extra.yellow[200],
    light: primitives.extra.yellow[100],
    dark: primitives.extra.yellow[300],
    contrast: primitives.brand.sunrise[900],
    surface: primitives.extra.yellow[300],
  },
  charts: {
    stone: primitives.brand.stone[200],
    lavender: primitives.brand.lavender[200],
    sky: primitives.brand.sky[200],
    aqua: primitives.brand.aqua[300],
    sunrise: primitives.brand.sunrise[300],
    sunset: primitives.brand.sunset[200],
    red: primitives.extra.red[200],
  },
  neutral: {
    main: primitives.brand.stone[50],
    light: primitives.base.white,
    dark: primitives.brand.stone[100],
    contrast: primitives.brand.stone[50],
    surface: "#F0F1F426",
  },
};

const PEAK_LIGHT_THEME = {
  ...primitives,
  ...basisLight,
};

const PEAK_DARK_THEME = {
  ...primitives,
  ...basisDark,
};

const pmmThemeOptions = (mode: PaletteMode): ThemeOptions => {
  const peakTheme = mode === "light" ? PEAK_LIGHT_THEME : PEAK_DARK_THEME;
  const newOptions: ThemeOptions = {
    palette: {
      mode,
      ...(mode === "light"
        ? {
            primary: {
              main: peakTheme.primary.main,
              dark: peakTheme.primary.dark,
              light: peakTheme.primary.light,
              contrastText: peakTheme.primary.contrast,
            },
            action: {
              hover: "rgba(220, 63, 0, 0.04)",
              hoverOpacity: 0.04,
              selected: "rgba(220, 63, 0, 0.08)",
              selectedOpacity: 0.08,
              focus: "rgba(220, 63, 0, 0.12)",
              focusOpacity: 0.12,
              focusVisible: "rgba(220, 48, 0, 0.3)",
              focusVisibleOpacity: 0.3,
              outlinedBorder: "rgba(220, 33, 0, 0.5)",
              outlinedBorderOpacity: 0.5,
            },
          }
        : {
            primary: {
              main: peakTheme.primary.main,
              dark: peakTheme.primary.dark,
              light: peakTheme.primary.light,
              contrastText: peakTheme.primary.contrast,
            },
            warning: {
              main: peakTheme.warning.main,
              contrastText: peakTheme.warning.dark,
            },
            action: {
              hover: "rgba(245, 106, 51, 0.08)",
              hoverOpacity: 0.08,
              selected: "rgba(245, 106, 51, 0.12)",
              selectedOpacity: 0.12,
              focus: "rgba(245, 106, 51, 0.15)",
              focusOpacity: 0.15,
              focusVisible: "rgba(245, 106, 51, 0.3)",
              focusVisibleOpacity: 0.3,
              outlinedBorder: "rgba(245, 106, 51, 0.5)",
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
              color: peakTheme.warning.contrast,
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
            color: peakTheme.warning.light,
            borderColor: peakTheme.warning.main,
          },
        },
      },
      MuiBadge: {
        styleOverrides: {
          colorWarning: {
            backgroundColor: peakTheme.warning.light,
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
              color: peakTheme.primary.main,
              backgroundColor: peakTheme.primary.hover,
            },

            [`.${iconButtonClasses.root}:focus`]: {
              color: peakTheme.primary.main,
              backgroundColor: peakTheme.primary.focus,
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
              color: peakTheme.primary.main,
              backgroundColor: peakTheme.primary.hover,

              [`.${listItemIconClasses.root}`]: {
                color: peakTheme.primary.main,
              },

              [`&.${listItemButtonClasses.selected}`]: {
                color: peakTheme.primary.main,
                backgroundColor: peakTheme.primary.selected,
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
            color: peakTheme.neutral.contrast,
            backgroundColor: peakTheme.neutral.main,
          }),
          arrow: () => ({
            color: peakTheme.neutral.main,
          }),
        },
      },
    },
  };

  return deepmerge<ThemeOptions>(baseThemeOptions(mode), newOptions);
};

export default pmmThemeOptions;
