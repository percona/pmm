import {
  ComponentsOverrides,
  createTheme,
  PaletteMode,
  ThemeOptions,
} from "@mui/material";
import { DatePickerToolbarClassKey } from "@mui/x-date-pickers/DatePicker";
import { MultiSectionDigitalClockClassKey } from "@mui/x-date-pickers";
import { outlinedInputClasses } from "@mui/material/OutlinedInput";

declare module "@mui/material/styles" {
  interface PaletteOptions {
    surfaces?: {
      default?: string;
      backdrop?: string;
      high?: string;
      low?: string;
    };
    dividers?: {
      divider?: string;
      dividerStrong?: string;
      dividerStronger?: string;
      contour?: string;
    };
    tooltips?: {
      color?: string;
      background?: string;
    };
  }

  interface Palette extends PaletteOptions {}
  interface SimplePaletteColorOptions {
    surface?: string;
    hover?: string;
    selected?: string;
    focus?: string;
    focusVisible?: string;
    outlinedBorder?: string;
  }

  interface PaletteColor {
    surface?: string;
    hover?: string;
    selected?: string;
    focus?: string;
    focusVisible?: string;
    outlinedBorder?: string;
  }

  interface TypeAction {
    focusVisible: string;
    focusVisibleOpacity: number;
    outlinedBorder: string;
    outlinedBorderOpacity: number;
  }

  interface TypographyVariants {
    sectionHeading: React.CSSProperties;
    subHead1: React.CSSProperties;
    subHead2: React.CSSProperties;
    helperText: React.CSSProperties;
    menuText: React.CSSProperties;
    inputText: React.CSSProperties;
    inputLabel: React.CSSProperties;
  }

  interface TypographyVariantsOptions {
    sectionHeading: React.CSSProperties;
    subHead1: React.CSSProperties;
    subHead2: React.CSSProperties;
    helperText: React.CSSProperties;
    menuText: React.CSSProperties;
    inputText: React.CSSProperties;
    inputLabel: React.CSSProperties;
  }

  interface ComponentNameToClassKey {
    MuiDateCalendar: DatePickerToolbarClassKey;
    MuiMultiSectionDigitalClock: MultiSectionDigitalClockClassKey;
  }

  interface Components<Theme = unknown> {
    MuiDateCalendar?: {
      styleOverrides?: ComponentsOverrides<Theme>["MuiDateCalendar"];
    };
    MuiMultiSectionDigitalClock?: {
      styleOverrides?: ComponentsOverrides<Theme>["MuiMultiSectionDigitalClock"];
    };
  }
}

declare module "@mui/material/Typography" {
  interface TypographyPropsVariantOverrides {
    sectionHeading: true;
    subHead1: true;
    subHead2: true;
    helperText: true;
    menuText: true;
    inputText: true;
    inputLabel: true;
  }
}

declare module "@mui/material/Paper" {
  interface PaperPropsVariantOverrides {
    grey: true;
  }
}

// Color Primitives — Shared across all themes
export const primitives = {
  brand: {
    sunset: {
      900: "#6B350B",
      800: "#8C450E",
      700: "#B55912",
      600: "#E87318",
      500: "#FF7E1A",
      400: "#FF9848",
      300: "#FFA966",
      200: "#FFC496",
      100: "#FFD7B8",
      50: "#FFF2E8",
    },
    fire: {
      900: "#661D00",
      800: "#852600",
      700: "#AC3100",
      600: "#DC3F00",
      500: "#F24500",
      400: "#F56A33",
      300: "#F68254",
      200: "#F9A98A",
      100: "#FBC5B0",
      50: "#FEECE6",
    },
    night: {
      900: "#060B23",
      800: "#080E2E",
      700: "#0A123B",
      600: "#0D184C",
      500: "#0E1A53",
      400: "#3E4875",
      300: "#5E668C",
      200: "#9096B0",
      100: "#B4B8CA",
      50: "#E7E8EE",
    },
    sky: {
      900: "#08386B",
      800: "#0B4A8C",
      700: "#0E5FB5",
      600: "#127AE8",
      500: "#1486FF",
      400: "#439EFF",
      300: "#62AEFF",
      200: "#93C7FF",
      100: "#B6D9FF",
      50: "#E8F3FF",
    },
    sunrise: {
      900: "#654B17",
      800: "#84621E",
      700: "#AA7F26",
      600: "#DAA331",
      500: "#F0B336",
      400: "#F3C25E",
      300: "#F5CC78",
      200: "#F8DCA3",
      100: "#FAE7C1",
      50: "#FEF7EB",
    },
    aqua: {
      900: "#14584B",
      800: "#1A7362",
      700: "#22947E",
      600: "#2CBEA2",
      500: "#30D1B2",
      400: "#59DAC1",
      300: "#74E0CB",
      200: "#A0EADC",
      100: "#BFF1E7",
      50: "#EAFAF7",
    },
    lavender: {
      900: "#472F64",
      800: "#5C3D83",
      700: "#774FAA",
      600: "#9965D9",
      500: "#A86FEF",
      400: "#B98CF2",
      300: "#C59FF4",
      200: "#D7BDF8",
      100: "#E4D2FA",
      50: "#F6F1FD",
    },
    stone: {
      900: "#2C323E",
      800: "#3A4151",
      700: "#4B5468",
      600: "#606C86",
      500: "#697793",
      400: "#8792A9",
      300: "#9BA4B7",
      200: "#BAC0CD",
      100: "#D1D5DE",
      50: "#F0F1F4",
    }
  },
  extra: {
    yellow: {
      900: "#70510A",
      800: "#9C7407",
      700: "#C89A04",
      600: "#F2C202",
      500: "#FED520",
      400: "#FFE047",
      300: "#FFE770",
      200: "#FFEE99",
      100: "#FFF5C2",
      50: "#FFFCEB",
    },
    red: {
      900: "#4D0000",
      800: "#700000",
      700: "#920000",
      600: "#B10810",
      500: "#CC352E",
      400: "#E2584D",
      300: "#F37C6F",
      200: "#FEA195",
      100: "#FFC6BE",
      50: "#FFECE9",
    },
    green: {
      900: "#002C1D",
      800: "#004430",
      700: "#005C45",
      600: "#00745B",
      500: "#008C71",
      400: "#00A489",
      300: "#51BAA2",
      200: "#85CFBC",
      100: "#B6E3D6",
      50: "#E7F6F1",
    }
  },
  base: {
    white: "#FFFFFF",
    black: "#000000",
  }
} as const;

export type Primitives = typeof primitives;

// Semantic color tokens - Light mode
export const semanticTokensLight = {
  surfaces: {
    elevation0: primitives.brand.stone[50],
    elevation1: primitives.base.white,
    backdrop: "rgba(0, 0, 0, 0.7)",
  },
  text: {
    primary: "#2C323E",
    secondary: "rgba(44, 50, 62, 0.7)",
    disabled: "rgba(44, 50, 62, 0.4)",
    sky: primitives.brand.sky[600],
    lavender: primitives.brand.lavender[600],
    aqua: primitives.brand.aqua[700],
  },
  lines: {
    divider: "rgba(44, 50, 62, 0.25)",
    dividerStrong: "rgba(44, 50, 62, 0.5)",
    contour: "rgba(0, 0, 0, 0.06)",
    dividerStronger: "#2C323E",
  },
  action: {
    hover: "rgba(44, 50, 62, 0.04)",
    disabled: "rgba(44, 50, 62, 0.12)",
    focus: "rgba(44, 50, 62, 0.12)",
  },
  error: {
    main: primitives.extra.red[600],
    light: primitives.extra.red[500],
    dark: primitives.extra.red[700],
    contrastText: primitives.extra.red[700],
    surface: primitives.extra.red[50],
  },
  success: {
    main: primitives.extra.green[600],
    light: primitives.extra.green[500],
    dark: primitives.extra.green[700],
    contrastText: primitives.extra.green[700],
    surface: primitives.extra.green[50],
  },
  informative: {
    main: primitives.brand.sky[700],
    light: primitives.brand.sky[600],
    dark: primitives.brand.sky[800],
    contrastText: primitives.brand.sky[800],
    surface: primitives.brand.sky[50],
  },
  warning: {
    main: primitives.extra.yellow[800],
    light: primitives.brand.sunrise[700],
    dark: primitives.brand.sunrise[900],
    contrastText: primitives.brand.sunrise[900],
    surface: primitives.extra.yellow[100],
  },
  neutral: {
    main: primitives.brand.stone[800],
    light: primitives.brand.stone[700],
    dark: primitives.brand.stone[900],
    contrastText: primitives.brand.stone[800],
    surface: "rgba(44, 50, 62, 0.12)", // Same as action.focus
  },
  charts: {
    stone: primitives.brand.stone[400],
    lavender: primitives.brand.lavender[500],
    sky: primitives.brand.sky[500],
    aqua: primitives.brand.aqua[600],
    sunrise: primitives.brand.sunrise[500],
    sunset: primitives.brand.sunset[400],
    red: primitives.extra.red[400],
  }
};

// Semantic color tokens — Dark mode
export const semanticTokensDark = {
  surfaces: {
    elevation0: primitives.brand.stone[800],
    elevation1: primitives.brand.stone[900],
    backdrop: "rgba(0, 0, 0, 0.7)",
  },
  text: {
    primary: "#FBFBFB",
    secondary: "rgba(251, 251, 251, 0.7)",
    disabled: "rgba(251, 251, 251, 0.4)",
    sky: primitives.brand.sky[200],
    lavender: primitives.brand.lavender[200],
    aqua: primitives.brand.aqua[300],
  },
  lines: {
    divider: "rgba(255, 255, 255, 0.25)",
    dividerStrong: "rgba(255, 255, 255, 0.5)",
    contour: "rgba(255, 255, 255, 0.08)",
    dividerStronger: "#FFFFFF",
  },
  action: {
    hover: "rgba(240, 241, 244, 0.08)",
    disabled: "rgba(240, 241, 244, 0.15)",
    focus: "rgba(240, 241, 244, 0.15)",
  },
  error: {
    main: primitives.extra.red[300],
    light: primitives.extra.red[200],
    dark: primitives.extra.red[400],
    contrastText: primitives.base.white,
    surface: primitives.extra.red[400],
  },
  success: {
    main: primitives.extra.green[400],
    light: primitives.extra.green[300],
    dark: primitives.extra.green[500],
    contrastText: primitives.base.white,
    surface: primitives.extra.green[500],
  },
  informative: {
    main: primitives.brand.sky[500],
    light: primitives.brand.sky[400],
    dark: primitives.brand.sky[600],
    contrastText: primitives.base.white,
    surface: primitives.brand.sky[600],
  },
  warning: {
    main: primitives.extra.yellow[200],
    light: primitives.extra.yellow[100],
    dark: primitives.extra.yellow[300],
    contrastText: primitives.brand.sunrise[900],
    surface: primitives.extra.yellow[300],
  },
  neutral: {
    main: primitives.brand.stone[50],
    light: primitives.base.white,
    dark: primitives.brand.stone[100],
    contrastText: primitives.brand.stone[50],
    surface: "rgba(240, 241, 244, 0.15)",  // Same as action.focus
  },
  charts: {
    stone: primitives.brand.stone[200],
    lavender: primitives.brand.lavender[200],
    sky: primitives.brand.sky[200],
    aqua: primitives.brand.aqua[300],
    sunrise: primitives.brand.sunrise[300],
    sunset: primitives.brand.sunset[200],
    red: primitives.extra.red[200],
  }
};

// Primary color tokens
export const defaultPrimaryLight = {
  main: primitives.brand.stone[900],
  dark: primitives.base.black,
  light: primitives.brand.stone[800],
  contrastText: primitives.base.white,
  hover: "rgba(32, 68, 147, 0.04)",
  selected: "rgba(32, 68, 147, 0.08)",
  focus: "rgba(32, 68, 147, 0.12)",
  focusVisible: "rgba(47, 74, 132, 0.3)",
  outlinedBorder: "rgba(61, 79, 118, 0.5)",
};
export const defaultPrimaryDark = {
  main: primitives.brand.stone[50],
  dark: primitives.brand.stone[100],
  light: primitives.base.white,
  contrastText: primitives.base.black,
  hover: "rgba(209, 213, 222, 0.08)",
  selected: "rgba(209, 213, 222, 0.12)",
  focus: "rgba(209, 213, 222, 0.15)",
  focusVisible: "rgba(209, 213, 222, 0.3)",
  outlinedBorder: "rgba(209, 213, 222, 0.5)",
};

export type SemanticTokens = typeof semanticTokensLight;
export type PrimaryTokens = typeof defaultPrimaryLight;

const BaseTheme = createTheme();

const baseThemeOptions = (mode: PaletteMode): ThemeOptions => {
  const tokens = mode === "light" ? semanticTokensLight : semanticTokensDark;
  const primary = mode === "light" ? defaultPrimaryLight : defaultPrimaryDark;

  return {
  palette: {
    mode,
    // Primary colors tokens (Default for Base - other themes can override this)
    primary: {
      main: primary.main,
      dark: primary.dark,
      light: primary.light,
      contrastText: primary.contrastText,
      hover: primary.hover,
      selected: primary.selected,
      focus: primary.focus,
      focusVisible: primary.focusVisible,
      outlinedBorder: primary.outlinedBorder,
    },
    // Text colors
    text: {
      primary: tokens.text.primary,
      secondary: tokens.text.secondary,
      disabled: tokens.text.disabled,
    },
    // Messaging colors tokens
    error: {
      main: tokens.error.main,
      light: tokens.error.light,
      dark: tokens.error.dark,
      contrastText: tokens.error.contrastText,
      surface: tokens.error.surface,
    },
    warning: {
      main: tokens.warning.main,
      light: tokens.warning.light,
      dark: tokens.warning.dark,
      contrastText: tokens.warning.contrastText,
      surface: tokens.warning.surface,
    },
    info: {
      main: tokens.informative.main,
      light: tokens.informative.light,
      dark: tokens.informative.dark,
      contrastText: tokens.informative.contrastText,
      surface: tokens.informative.surface,
    },
    success: {
      main: tokens.success.main,
      light: tokens.success.light,
      dark: tokens.success.dark,
      contrastText: tokens.success.contrastText,
      surface: tokens.success.surface,
    },
    // Action colors
    action: {
      hover: tokens.action.hover,
      hoverOpacity: 0.04,
      disabled: tokens.action.disabled,
      disabledOpacity: 0.12,
      focus: tokens.action.focus,
      focusOpacity: 0.12,
    },
    // Background/Surfaces
    background: {
      default: tokens.surfaces.elevation0,
      paper: tokens.surfaces.elevation1,
    },
    surfaces: {
      backdrop: tokens.surfaces.backdrop,
      low: tokens.surfaces.elevation0,
      high: tokens.surfaces.elevation1,
    },
    // Other color tokens
    dividers: {
      divider: tokens.lines.divider,
      dividerStrong: tokens.lines.dividerStrong,
      dividerStronger: tokens.lines.dividerStronger,
      contour: tokens.lines.contour,
    },
    tooltips: {
      background: tokens.neutral.main,
      color: tokens.neutral.contrastText,
    },
  },
  typography: {
    h1: {
      fontWeight: 600,
      fontFamily: "'Poppins', sans-serif",
      [BaseTheme.breakpoints.down("sm")]: {
        fontSize: "32px",
      },
      [BaseTheme.breakpoints.up("sm")]: {
        fontSize: "48px",
      },
    },
    h2: {
      fontWeight: 600,
      fontFamily: "'Poppins', sans-serif",
      [BaseTheme.breakpoints.down("sm")]: {
        fontSize: "29px",
      },
      [BaseTheme.breakpoints.up("sm")]: {
        fontSize: "40px",
      },
    },
    h3: {
      fontWeight: 600,
      fontFamily: "'Poppins', sans-serif",
      [BaseTheme.breakpoints.down("sm")]: {
        fontSize: "26px",
      },
      [BaseTheme.breakpoints.up("sm")]: {
        fontSize: "33px",
      },
    },
    h4: {
      fontWeight: 600,
      fontFamily: "'Poppins', sans-serif",
      [BaseTheme.breakpoints.down("sm")]: {
        fontSize: "23px",
      },
      [BaseTheme.breakpoints.up("sm")]: {
        fontSize: "28px",
      },
    },
    h5: {
      fontWeight: 600,
      fontFamily: "'Poppins', sans-serif",
      lineHeight: "22.5px",
      [BaseTheme.breakpoints.down("sm")]: {
        fontSize: "20px",
      },
      [BaseTheme.breakpoints.up("sm")]: {
        fontSize: "23px",
      },
    },
    h6: {
      fontWeight: 600,
      fontFamily: "'Poppins', sans-serif",
      [BaseTheme.breakpoints.down("sm")]: {
        fontSize: "18px",
      },
      [BaseTheme.breakpoints.up("sm")]: {
        fontSize: "19px",
      },
    },
    subtitle1: {
      fontWeight: 600,
      fontSize: "19px",
      lineHeight: "1",
    },
    subtitle2: {
      fontWeight: 600,
      fontSize: "16px",
      lineHeight: "22px",
    },
    overline: {
      fontFamily: "'Poppins', sans-serif",
      fontWeight: 800,
      fontSize: "12px",
    },
    sectionHeading: {
      fontFamily: "'Poppins', sans-serif",
      fontWeight: 700,
      fontSize: "14px",
      lineHeight: "17.5px",
    },
    subHead1: {
      fontWeight: 600,
      fontSize: "19px",
      lineHeight: "26.13px",
    },
    subHead2: {
      fontWeight: 600,
      fontSize: "16px",
      lineHeight: "22px",
    },
    helperText: {
      fontWeight: 450,
      fontSize: "12px",
      lineHeight: "15px",
    },
    menuText: {
      fontWeight: 500,
      fontSize: "16px",
      lineHeight: "20px",
    },
    body1: {
      fontWeight: 400,
      fontSize: "16px",
    },
    body2: {
      fontWeight: 400,
      fontSize: "14px",
    },
    caption: {
      fontWeight: 400,
      fontSize: "13px",
    },
    button: {
      fontFamily: "'Poppins', sans-serif",
      textTransform: "none",
      lineHeight: "1",
      letterSpacing: "0.025em",
    },
    inputText: {
      fontSize: "16px",
      fontWeight: 400,
    },
    inputLabel: {
      fontSize: "12px",
      fontWeight: 500,
    },
  },
  shadows: [
    "none",
    "0px 2px 1px -1px rgba(0, 0, 0, 0.2), 0px 1px 1px 0px rgba(0, 0, 0, 0.14), 0px 1px 3px 0px rgba(0, 0, 0, 0.12)",
    "0px 3px 1px -2px rgba(0, 0, 0, 0.2), 0px 2px 2px rgba(0, 0, 0, 0.14), 0px 1px 5px rgba(0, 0, 0, 0.12)",
    "0px 3px 3px -2px rgba(0, 0, 0, 0.2), 0px 3px 4px 0px rgba(0, 0, 0, 0.14), 0px 1px 8px 0px rgba(0, 0, 0, 0.12)",
    "0px 2px 4px -1px rgba(0, 0, 0, 0.2), 0px 4px 5px rgba(0, 0, 0, 0.14), 0px 1px 10px rgba(0, 0, 0, 0.12)",
    "0px 3px 5px -1px rgba(0, 0, 0, 0.2), 0px 5px 8px 0px rgba(0, 0, 0, 0.14), 0px 1px 14px 0px rgba(0, 0, 0, 0.12)",
    "0px 3px 5px -1px rgba(0, 0, 0, 0.2), 0px 6px 10px 0px rgba(0, 0, 0, 0.14), 0px 1px 18px 0px rgba(0, 0, 0, 0.12)",
    "0px 4px 5px -2px rgba(0, 0, 0, 0.2), 0px 7px 10px 1px rgba(0, 0, 0, 0.14), 0px 2px 16px 1px rgba(0, 0, 0, 0.12)",
    "0px 5px 5px -3px rgba(0, 0, 0, 0.2), 0px 8px 10px 1px rgba(0, 0, 0, 0.14), 0px 3px 14px 2px rgba(0, 0, 0, 0.12)",
    "0px 5px 6px -3px rgba(0, 0, 0, 0.2), 0px 9px 12px 1px rgba(0, 0, 0, 0.14), 0px 3px 16px 2px rgba(0, 0, 0, 0.12)",
    "0px 6px 6px -3px rgba(0, 0, 0, 0.2), 0px 10px 14px 1px rgba(0, 0, 0, 0.14), 0px 4px 18px 3px rgba(0, 0, 0, 0.12)",
    "0px 6px 7px -4px rgba(0, 0, 0, 0.2), 0px 11px 15px 1px rgba(0, 0, 0, 0.14), 0px 4px 20px 3px rgba(0, 0, 0, 0.12)",
    "0px 7px 8px -4px rgba(0, 0, 0, 0.2), 0px 12px 17px 2px rgba(0, 0, 0, 0.14), 0px 5px 22px 4px rgba(0, 0, 0, 0.12)",
    "0px 7px 8px -4px rgba(0, 0, 0, 0.2), 0px 13px 19px 2px rgba(0, 0, 0, 0.14), 0px 5px 24px 4px rgba(0, 0, 0, 0.12)",
    "0px 7px 9px -4px rgba(0, 0, 0, 0.2), 0px 14px 21px 2px rgba(0, 0, 0, 0.14), 0px 5px 26px 4px rgba(0, 0, 0, 0.12)",
    "0px 8px 9px -5px rgba(0, 0, 0, 0.2), 0px 15px 22px 2px rgba(0, 0, 0, 0.14), 0px 6px 28px 5px rgba(0, 0, 0, 0.12)",
    "0px 8px 10px -5px rgba(0, 0, 0, 0.2), 0px 16px 24px 2px rgba(0, 0, 0, 0.14), 0px 6px 30px 5px rgba(0, 0, 0, 0.12)",
    "0px 8px 11px -5px rgba(0, 0, 0, 0.2), 0px 17px 26px 2px rgba(0, 0, 0, 0.14), 0px 6px 32px 5px rgba(0, 0, 0, 0.12)",
    "0px 9px 11px -5px rgba(0, 0, 0, 0.2), 0px 18px 28px 2px rgba(0, 0, 0, 0.14), 0px 7px 34px 6px rgba(0, 0, 0, 0.12)",
    "0px 9px 12px -6px rgba(0, 0, 0, 0.2), 0px 19px 29px 2px rgba(0, 0, 0, 0.14), 0px 7px 36px 6px rgba(0, 0, 0, 0.12)",
    "0px 10px 13px -6px rgba(0, 0, 0, 0.2), 0px 20px 31px 3px rgba(0, 0, 0, 0.14), 0px 8px 38px 7px rgba(0, 0, 0, 0.12)",
    "0px 10px 13px -6px rgba(0, 0, 0, 0.2), 0px 21px 33px 3px rgba(0, 0, 0, 0.14), 0px 8px 40px 7px rgba(0, 0, 0, 0.12)",
    "0px 10px 14px -6px rgba(0, 0, 0, 0.2), 0px 22px 35px 3px rgba(0, 0, 0, 0.14), 0px 8px 42px 7px rgba(0, 0, 0, 0.12)",
    "0px 11px 14px -7px rgba(0, 0, 0, 0.2), 0px 23px 36px 3px rgba(0, 0, 0, 0.14), 0px 9px 44px 8px rgba(0, 0, 0, 0.12)",
    "0px 11px 15px -7px rgba(0, 0, 0, 0.2), 0px 24px 38px 3px rgba(0, 0, 0, 0.14), 0px 9px 46px 8px rgba(0, 0, 0, 0.12)",
  ],
  breakpoints: {
    values: {
      xs: 0,
      sm: 600,
      md: 900,
      lg: 1200,
      xl: 1536,
    },
  },
  components: {
    MuiInputBase: {
      styleOverrides: {
        root: () => ({
          fontSize: "16px",
          fontWeight: 400,
          maxWidth: "100%",
        }),
      },
    },
    MuiButtonBase: {
      defaultProps: {
        disableRipple: true,
      },
      styleOverrides: {
        root: {
          "&.MuiIconButton-root:not(:focus-visible):focus": {
            backgroundColor: "transparent",
          },
          "&.MuiIconButton-root:focus-visible:focus": {
            backgroundColor: "rgba(18, 119, 227, 0.3)",
          },
          "&.MuiIconButton-root:focus-visible": {
            backgroundColor: "rgba(18, 119, 227, 0.3)",
          },
        },
      },
    },
    MuiButton: {
      defaultProps: {
        disableTouchRipple: true,
      },
      styleOverrides: {
        root: ({ ownerState, theme }) => ({
          borderRadius: 128,
          borderWidth: 2,

          ".MuiButton-startIcon": {
            height: 0,
            alignItems: "center",
          },

          ...(ownerState.variant === "contained" && {
            ...(ownerState.size === "large" && {
              padding: "12px 24px",
            }),
            ...(ownerState.size === "medium" && {
              padding: "11px 16px",
            }),
            ...(ownerState.size === "small" && {
              padding: "8px 12px",
            }),
          }),

          ...(ownerState.variant === "outlined" && {
            ...(ownerState.size === "large" && {
              padding: "12px 22px",
            }),
            ...(ownerState.size === "medium" && {
              padding: "11px 16px",
            }),
            ...(ownerState.size === "small" && {
              padding: "8px 10px",
            }),
            borderColor: theme.palette.primary.main,
          }),

          ...(ownerState.variant === "text" && {
            ...(ownerState.size === "large" && {
              padding: "8px 11px",
            }),
            ...(ownerState.size === "medium" && {
              padding: "6px 8px",
            }),
            ...(ownerState.size === "small" && {
              padding: "4px 5px",
            }),
            borderColor: theme.palette.primary.main,
          }),

          ...(ownerState.size === "large" && {
            fontSize: 15,
          }),
          ...(ownerState.size === "medium" && {
            fontSize: 13,
          }),
          ...(ownerState.size === "small" && {
            fontSize: 13,
          }),

          "&:hover": {
            borderWidth: "2px",
            ...(ownerState.variant === "outlined" && {
              backgroundColor: theme.palette.action.focus,
            }),
            ...(ownerState.variant === "text" && {
              backgroundColor: theme.palette.action.focus,
            }),
          },
          "&:disabled": {
            borderWidth: "2px",
            ...(ownerState.variant === "contained" && {
              backgroundColor: theme.palette.action.disabled,
            }),
            color: theme.palette.text.disabled,
          },
        }),
      },
    },
    MuiButtonGroup: {
      styleOverrides: {
        root: {
          borderRadius: "128px",
        },
        grouped: ({ ownerState }) => ({
          "&:not(:last-of-type)": {
            ...(ownerState.variant === "contained" && {
              borderRight: 0,
            }),
            marginLeft: "-2px",
          },
          "&:not(:first-of-type)": {
            marginLeft: "-2px",
          },
        }),
      },
    },
    MuiMenuItem: {
      styleOverrides: {
        root: ({ theme }) => ({
          ...theme.typography.menuText,
          // This should override any other nested typography (e.g. form labels)
          ".MuiTypography-root": {
            ...theme.typography.menuText,
          },
          "&.Mui-disabled": {
            opacity: 0.5,
          },
        }),
      },
    },
    MuiOutlinedInput: {
      styleOverrides: {
        root: ({ theme }) => ({
          ...theme.typography.inputText,
          "& fieldset": {
            borderColor: theme.palette.dividers?.divider,
          },
          [`&:hover .${outlinedInputClasses.notchedOutline}`]: {
            borderColor: theme.palette.dividers?.dividerStrong,
          },
          [`&.Mui-focused .${outlinedInputClasses.notchedOutline}`]: {
            borderColor: theme.palette.dividers?.dividerStronger,
          },
          [`&:disabled .${outlinedInputClasses.notchedOutline}`]: {
            borderColor: theme.palette.dividers?.contour,
          },
        }),
      },
    },
    MuiInputLabel: {
      styleOverrides: {
        root: ({ theme }) => ({
          ...theme.typography.inputText,
        }),
      },
    },
    MuiFormGroup: {
      styleOverrides: {
        root: () => ({
          "> *:first-child": {
            marginTop: 0,
          },
        }),
      },
    },
    MuiFormControl: {
      styleOverrides: {
        root: ({ theme }) => ({
          ".MuiInputLabel-root": {
            color: theme.palette.text.secondary,
            "&.Mui-focused": {
              color: theme.palette.text.primary,
            },
          },
        }),
      },
    },
    MuiFormHelperText: {
      styleOverrides: {
        root: {
          fontSize: "12px",
          fontWeight: 400,
        },
      },
    },
    MuiTabs: {
      styleOverrides: {
        root: {
          minHeight: "32px",
          ".MuiButtonBase-root": {
            fontSize: "16px",
            padding: "8px 12px",
          },
          ".MuiTabs-indicator": {
            height: "3px",
          },
          ".MuiTabScrollButton-root": {
            "&.Mui-disabled": {
              opacity: 0.1,
            },
          },
        },
      },
    },
    MuiDialogTitle: {
      defaultProps: {
        component: "h5",
        variant: "h5",
      },
      styleOverrides: {
        root: {
          padding: BaseTheme.spacing(2),
        },
      },
    },
    MuiDialogContent: {
      styleOverrides: {
        root: {
          padding: BaseTheme.spacing(2),
          paddingTop: `${BaseTheme.spacing(2)} !important`,
        },
      },
    },
    MuiDialogActions: {
      styleOverrides: {
        root: {
          padding: BaseTheme.spacing(2),
        },
      },
    },
    MuiTooltip: {
      styleOverrides: {
        tooltip: ({ theme }) => ({
          ...theme.typography.helperText,
          boxShadow: theme.shadows[8],
          ...(theme.palette.tooltips && {
            color: theme.palette.tooltips?.color,
            backgroundColor: theme.palette.tooltips?.background,
          }),
        }),
        arrow: ({ theme }) => ({
          ...(theme.palette.tooltips && {
            color: theme.palette.tooltips?.background,
          }),
        }),
      },
    },
    MuiAccordion: {
      styleOverrides: {
        root: ({ theme }) => ({
          borderWidth: "1px",
          borderStyle: "solid",
          borderColor: "rgba(44, 50, 62, 0.25)", // TODO move into pallet =#2C323E 25%
          borderRadius: theme.spacing(1),
          backgroundColor: theme.palette.action.hover,
          boxShadow: "none",
          "&:before": {
            height: 0,
          },
          marginTop: "8px",
          "&.Mui-expanded": {
            marginTop: "8px",
          },
        }),
      },
    },
    MuiAccordionSummary: {
      styleOverrides: {
        root: () => ({
          "&.Mui-disabled": {
            opacity: "0.5 !important",
          },
        }),
      },
    },
    MuiCard: {
      variants: [
        {
          props: { variant: "grey" },
          style: ({ theme }) => ({
            border: `${theme.palette.dividers?.divider} 1px solid`,
            ".MuiCardHeader-root": {
              backgroundColor: theme.palette.surfaces?.low,
              borderBottomColor: theme.palette.dividers?.divider,
              borderBottomWidth: 1,
              borderBottomStyle: "solid",
              padding: `${theme.spacing(1.5)} ${theme.spacing(2)}`,
              ".MuiTypography-root": {
                ...theme.typography.sectionHeading,
              },
              "#database-icon, #network-node-icon": {
                path: { fill: theme.palette.text.primary },
              },
            },
            ".MuiCardContent": {
              py: 1,
              px: 2,
            },
          }),
        },
      ],
    },
    MuiDateCalendar: {
      styleOverrides: {
        root: ({ theme }) => ({
          ".MuiPickersCalendarHeader-root": {
            marginTop: theme.spacing(2),
            marginBottom: theme.spacing(2),
          },
          ".MuiPickersCalendarHeader-label": {
            fontSize: theme.typography.inputText.fontSize,
            fontWeight: theme.typography.inputText.fontWeight,
          },
          ".MuiDayCalendar-weekDayLabel": {
            fontSize: theme.typography.helperText.fontSize,
            fontWeight: theme.typography.helperText.fontWeight,
            color: theme.palette.text.disabled,
          },
          ".MuiPickersDay-root": {
            fontSize: theme.typography.body2.fontSize,
            fontWeight: theme.typography.body2.fontWeight,
          },
        }),
      },
    },
    MuiMultiSectionDigitalClock: {
      styleOverrides: {
        root: () => ({
          ".MuiMultiSectionDigitalClockSection-root": {
            "&::after": {
              height: "calc(100% - 40px - 2px)",
            },
          },
        }),
      },
    },
    MuiAlertTitle: {
      styleOverrides: {
        root: ({ theme }) => ({
          ...theme.typography.h6,
        }),
      },
    },
    MuiAlert: {
      styleOverrides: {
        root: ({ theme, ownerState: { severity } }) => ({
          ...theme.typography.body2,
          borderRadius: "4px",
          borderWidth: "1px",
          borderStyle: "solid",
          borderColor:
            theme.palette.mode === "light"
              ? "rgba(0, 0, 0, 0.059)"
              : "transparent",
          backgroundColor: theme.palette[severity!].surface,
        }),
        icon: ({ theme, ownerState: { severity } }) => ({
          color: `${theme.palette[severity!].contrastText} !important`,
        }),
        message: ({ theme, ownerState: { severity } }) => ({
          color: theme.palette[severity!].contrastText,
        }),
      },
    },
    MuiChip: {
      styleOverrides: {
        filled: ({ theme, ownerState: { color } }) => ({
          // @ts-ignore
          backgroundColor: theme.palette[color]?.surface,
        }),
      },
    },
    MuiTableHead: {
      styleOverrides: {
        root: () => ({
          ".MuiBadge-overlapCircular": {
            backgroundColor: "transparent",
          },
        }),
      },
    },

    MuiTableBody: {
      styleOverrides: {
        root: ({ theme }) => ({
          "#empty-state-icon": {
            path: {
              // complex selector we need in order to provide dark theme style for this icon, instead of having a separate one
              "&:not(:nth-child(n+8)), &:last-child": {
                stroke: theme.palette.text.primary,
              },
            },
          },
        }),
      },
    },
  },
};
};

export default baseThemeOptions;
