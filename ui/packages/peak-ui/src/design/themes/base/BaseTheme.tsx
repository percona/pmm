import { ComponentsOverrides, createTheme, PaletteMode, ThemeOptions } from '@mui/material';
import { DatePickerToolbarClassKey } from '@mui/x-date-pickers/DatePicker';
import { MultiSectionDigitalClockClassKey } from '@mui/x-date-pickers';
import { outlinedInputClasses } from '@mui/material/OutlinedInput';

declare module '@mui/material/styles' {
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
      styleOverrides?: ComponentsOverrides<Theme>['MuiDateCalendar'];
    };
    MuiMultiSectionDigitalClock?: {
      styleOverrides?: ComponentsOverrides<Theme>['MuiMultiSectionDigitalClock'];
    };
  }
}

declare module '@mui/material/Typography' {
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

declare module '@mui/material/Paper' {
  interface PaperPropsVariantOverrides {
    grey: true;
  }
}

const fontDisplay =
  "'Poppins', -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Helvetica Neue', Arial, sans-serif";

// Color Primitives — Shared across all themes (Figma: primary / tech / extra / pure)
export const primitives = {
  primary: {
    purple: {
      950: '#1F0D68',
      900: '#381999',
      800: '#421CBB',
      700: '#5022DF',
      600: '#653DF4',
      500: '#7056FC',
      400: '#9387FE',
      300: '#B6B2FF',
      200: '#D6D5FF',
      100: '#EAE8FF',
      50: '#F3F3FF',
    },
    yellow: {
      950: '#2E3003',
      900: '#5A5F02',
      800: '#7D8301',
      700: '#9FA701',
      600: '#B8C200',
      500: '#D4DB00',
      400: '#E8F000',
      300: '#F6FE54',
      200: '#F9FDA5',
      100: '#FBFDD3',
      50: '#FCFDE7',
    },
    black: {
      950: '#282727',
      900: '#3D3C3C',
      800: '#464444',
      700: '#504E4F',
      600: '#5E5C5D',
      500: '#6F6B6B',
      400: '#8A8686',
      300: '#B1AFB0',
      200: '#D1D0D0',
      100: '#E7E6E6',
      50: '#F6F5F5',
    },
  },
  tech: {
    mysql: {
      950: '#411409',
      900: '#782E16',
      800: '#953517',
      700: '#BB4213',
      600: '#E65A15',
      500: '#F0721F',
      400: '#F49143',
      300: '#F8B879',
      200: '#FBD6AD',
      100: '#FDEDD7',
      50: '#FEF7EE',
    },
    redis: {
      950: '#43100C',
      900: '#7C2620',
      800: '#95271F',
      700: '#B42B21',
      600: '#D6362A',
      500: '#EA5449',
      400: '#F47E75',
      300: '#F9AEA8',
      200: '#FCCFCC',
      100: '#FDE5E3',
      50: '#FEF3F2',
    },
    mongodb: {
      950: '#072C10',
      900: '#175025',
      800: '#1A6129',
      700: '#1B7A2F',
      600: '#1FA23A',
      500: '#2BBC48',
      400: '#52D66D',
      300: '#8CE99F',
      200: '#BEF4C8',
      100: '#DEFAE3',
      50: '#F1FCF2',
    },
    postgresql: {
      950: '#0A3161',
      900: '#084AA0',
      800: '#005ED6',
      700: '#0073FF',
      600: '#2486FF',
      500: '#479DFF',
      400: '#6BB0FF',
      300: '#8FC5FF',
      200: '#ADD5FF',
      100: '#CCE5FF',
      50: '#EBF5FF',
    },
    valkey: {
      950: '#420566',
      900: '#601A89',
      800: '#751EAB',
      700: '#8B1FD1',
      600: '#A83FEF',
      500: '#B452FA',
      400: '#C982FE',
      300: '#DDB3FF',
      200: '#ECD4FF',
      100: '#F5E8FF',
      50: '#FBF5FF',
    },
    kubernetes: {
      950: '#0D2E44',
      900: '#144866',
      800: '#12567A',
      700: '#106494',
      600: '#127EB7',
      500: '#2AA6DF',
      400: '#48B6E8',
      300: '#88CFF1',
      200: '#C0E4F7',
      100: '#E3F1FB',
      50: '#F1F9FE',
    },
  },
  extra: {
    gold: {
      950: '#150F04',
      900: '#493408',
      800: '#7D5A08',
      700: '#AC8006',
      600: '#D7A604',
      500: '#FCCA03',
      400: '#FED520',
      300: '#FFDF3D',
      200: '#FFE866',
      100: '#FFF199',
      50: '#FFFBE0',
    },
  },
  pure: {
    black: '#000000',
    white: '#FFFFFF',
  },
} as const;

export type Primitives = typeof primitives;

// Semantic color tokens - Light mode
export const semanticTokensLight = {
  // Structural UI (light mode)
  surfaces: {
    elevation0: primitives.primary.black[50],
    elevation1: primitives.pure.white,
    backdrop: 'rgba(0, 0, 0, 0.7)',
  },
  text: {
    primary: 'rgb(40, 39, 39)',
    secondary: 'rgba(40, 39, 39, 0.7)',
    disabled: 'rgba(40, 39, 39, 0.4)',
    accent1: primitives.tech.kubernetes[600],
    accent2: primitives.primary.purple[500],
    accent3: primitives.primary.yellow[800],
  },
  lines: {
    contour: 'rgba(0, 0, 0, 0.08)',
    divider: 'rgba(40, 39, 39, 0.25)',
    dividerStrong: 'rgba(40, 39, 39, 0.5)',
    dividerStronger: 'rgb(40, 39, 39)',
  },
  action: {
    hover: 'rgba(40, 39, 39, 0.04)',
    disabled: 'rgba(40, 39, 39, 0.12)',
    focus: 'rgba(40, 39, 39, 0.12)',
  },
  // Semantic (light mode)
  success: {
    dark: primitives.tech.mongodb[900],
    main: primitives.tech.mongodb[700],
    light: primitives.tech.mongodb[500],
    contrastText: primitives.tech.mongodb[900],
    surface: primitives.tech.mongodb[50],
  },
  informative: {
    dark: primitives.tech.postgresql[900],
    main: primitives.tech.postgresql[700],
    light: primitives.tech.postgresql[500],
    contrastText: primitives.tech.postgresql[900],
    surface: primitives.tech.postgresql[50],
  },
  warning: {
    dark: primitives.extra.gold[800],
    main: primitives.extra.gold[600],
    light: primitives.extra.gold[400],
    contrastText: primitives.extra.gold[800],
    surface: primitives.extra.gold[50],
  },
  error: {
    dark: primitives.tech.redis[900],
    main: primitives.tech.redis[700],
    light: primitives.tech.redis[500],
    contrastText: primitives.tech.redis[900],
    surface: primitives.tech.redis[50],
  },
  neutral: {
    dark: primitives.primary.black[950],
    main: primitives.primary.black[700],
    light: primitives.primary.black[400],
    contrastText: primitives.primary.black[950],
    surface: 'rgba(40, 39, 39, 0.12)',
  },
  // Charts (light mode)
  charts: {
    chart1: primitives.primary.purple[400],
    chart2: primitives.primary.yellow[400],
    chart3: primitives.tech.postgresql[500],
    chart4: primitives.tech.mysql[400],
    chart5: primitives.tech.mongodb[400],
    chart6: primitives.tech.redis[500],
    chart7: primitives.primary.black[300],
  },
};

// Semantic color tokens — Dark mode
export const semanticTokensDark = {
  // Structural UI (dark mode)
  surfaces: {
    elevation0: primitives.primary.black[900],
    elevation1: primitives.primary.black[950],
    backdrop: 'rgba(0, 0, 0, 0.7)',
  },
  text: {
    primary: 'rgb(246, 245, 245)',
    secondary: 'rgba(246, 245, 245, 0.7)',
    disabled: 'rgba(246, 245, 245, 0.4)',
    accent1: primitives.tech.kubernetes[300],
    accent2: primitives.primary.purple[300],
    accent3: primitives.primary.yellow[200],
  },
  lines: {
    contour: 'rgba(255, 255, 255, 0.12)',
    divider: 'rgba(255, 255, 255, 0.25)',
    dividerStrong: 'rgba(255, 255, 255, 0.5)',
    dividerStronger: 'rgb(255, 255, 255)',
  },
  action: {
    hover: 'rgba(246, 245, 245, 0.08)',
    disabled: 'rgba(246, 245, 245, 0.15)',
    focus: 'rgba(246, 245, 245, 0.15)',
  },
  // Semantic (dark mode)
  success: {
    dark: primitives.tech.mongodb[800],
    main: primitives.tech.mongodb[600],
    light: primitives.tech.mongodb[400],
    contrastText: primitives.pure.white,
    surface: primitives.tech.mongodb[800],
  },
  informative: {
    dark: primitives.tech.postgresql[700],
    main: primitives.tech.postgresql[500],
    light: primitives.tech.postgresql[300],
    contrastText: primitives.pure.white,
    surface: primitives.tech.postgresql[700],
  },
  warning: {
    dark: primitives.extra.gold[600],
    main: primitives.extra.gold[400],
    light: primitives.extra.gold[200],
    contrastText: primitives.extra.gold[900],
    surface: primitives.extra.gold[100],
  },
  error: {
    dark: primitives.tech.redis[700],
    main: primitives.tech.redis[500],
    light: primitives.tech.redis[300],
    contrastText: primitives.pure.white,
    surface: primitives.tech.redis[700],
  },
  neutral: {
    dark: primitives.primary.black[600],
    main: primitives.primary.black[400],
    light: primitives.primary.black[200],
    contrastText: primitives.primary.black[50],
    surface: 'rgba(246, 245, 245, 0.15)',
  },
  // Charts (dark mode)
  charts: {
    chart1: primitives.primary.purple[400],
    chart2: primitives.primary.yellow[300],
    chart3: primitives.tech.postgresql[400],
    chart4: primitives.tech.mysql[300],
    chart5: primitives.tech.mongodb[300],
    chart6: primitives.tech.redis[400],
    chart7: primitives.primary.black[300],
  },
};

// Primary color tokens
export const defaultPrimaryLight = {
  dark: primitives.primary.black[950],
  main: primitives.primary.black[900],
  light: primitives.primary.black[800],
  contrastText: primitives.pure.white,
  hover: 'rgba(40, 39, 39, 0.04)',
  selected: 'rgba(40, 39, 39, 0.08)',
  focus: 'rgba(40, 39, 39, 0.12)',
  focusVisible: 'rgba(40, 39, 39, 0.3)',
  outlinedBorder: 'rgba(40, 39, 39, 0.5)',
};
export const defaultPrimaryDark = {
  dark: primitives.primary.black[50],
  main: primitives.pure.white,
  light: primitives.pure.white,
  contrastText: primitives.primary.black[950],
  hover: 'rgba(255, 255, 255, 0.08)',
  selected: 'rgba(255, 255, 255, 0.12)',
  focus: 'rgba(255, 255, 255, 0.15)',
  focusVisible: 'rgba(255, 255, 255, 0.3)',
  outlinedBorder: 'rgba(255, 255, 255, 0.5)',
};

export type SemanticTokens = typeof semanticTokensLight;
export type PrimaryTokens = typeof defaultPrimaryLight;

const BaseTheme = createTheme();

const baseThemeOptions = (mode: PaletteMode): ThemeOptions => {
  const tokens = mode === 'light' ? semanticTokensLight : semanticTokensDark;
  const primary = mode === 'light' ? defaultPrimaryLight : defaultPrimaryDark;

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
        fontFamily: fontDisplay,
        fontWeight: 600,
        [BaseTheme.breakpoints.down('sm')]: {
          fontSize: '2rem',
          letterSpacing: '-0.04rem',
        },
        [BaseTheme.breakpoints.up('sm')]: {
          fontSize: '3rem',
          letterSpacing: '-0.06rem',
        },
        lineHeight: '1.0625',
      },
      h2: {
        fontFamily: fontDisplay,
        fontWeight: 600,
        [BaseTheme.breakpoints.down('sm')]: {
          fontSize: '1.8125rem',
          letterSpacing: '-0.01813rem',
        },
        [BaseTheme.breakpoints.up('sm')]: {
          fontSize: '2.5rem',
          letterSpacing: '-0.025rem',
        },
        lineHeight: '1.0625',
      },
      h3: {
        fontFamily: fontDisplay,
        fontWeight: 600,
        [BaseTheme.breakpoints.down('sm')]: {
          fontSize: '1.625rem',
        },
        [BaseTheme.breakpoints.up('sm')]: {
          fontSize: '2.0625rem',
        },
        lineHeight: '1.0625',
      },
      h4: {
        fontFamily: fontDisplay,
        fontWeight: 600,
        [BaseTheme.breakpoints.down('sm')]: {
          fontSize: '1.4375rem',
        },
        [BaseTheme.breakpoints.up('sm')]: {
          fontSize: '1.75rem',
        },
        lineHeight: '1.0625',
      },
      h5: {
        fontFamily: fontDisplay,
        fontWeight: 600,
        [BaseTheme.breakpoints.down('sm')]: {
          fontSize: '1.25rem',
        },
        [BaseTheme.breakpoints.up('sm')]: {
          fontSize: '1.4375rem',
        },
        lineHeight: '1.125',
      },
      h6: {
        fontFamily: fontDisplay,
        fontWeight: 600,
        [BaseTheme.breakpoints.down('sm')]: {
          fontSize: '1.125rem',
        },
        [BaseTheme.breakpoints.up('sm')]: {
          fontSize: '1.1875rem',
        },
        lineHeight: '1.25',
      },
      subHead1: {
        fontWeight: 500,
        fontSize: '1.1875rem',
        lineHeight: '1.125',
      },
      subHead2: {
        fontWeight: 500,
        fontSize: '1rem',
        lineHeight: '1.25',
      },
      overline: {
        fontFamily: fontDisplay,
        fontWeight: 700,
        fontSize: '0.75rem',
        lineHeight: '1.125',
        letterSpacing: '0.03rem',
        textTransform: 'uppercase',
      },
      sectionHeading: {
        fontFamily: fontDisplay,
        fontWeight: 700,
        fontSize: '0.875rem',
        lineHeight: '1.25',
        letterSpacing: '0.00875rem',
      },
      body1: {
        fontWeight: 400,
        fontSize: '1rem',
        lineHeight: '1.375',
      },
      body2: {
        fontWeight: 400,
        fontSize: '0.875rem',
        lineHeight: '1.5',
      },
      caption: {
        fontWeight: 400,
        fontSize: '0.8125rem',
        lineHeight: '1.375',
      },
      button: {
        fontWeight: 600,
        fontFamily: fontDisplay,
        fontSize: '0.9375rem',
        lineHeight: '1.0625',
        textTransform: 'none',
      },
      menuText: {
        fontFamily: fontDisplay,
        fontWeight: 600,
        fontSize: '0.875rem',
        lineHeight: '1.25',
      },
      inputText: {
        fontWeight: 400,
        fontSize: '1rem',
        lineHeight: '1.5',
      },
      inputLabel: {
        fontWeight: 500,
        fontSize: '0.75rem',
        lineHeight: '1',
        letterSpacing: '0.0075rem',
      },
      helperText: {
        fontWeight: 450,
        fontSize: '0.75rem',
        lineHeight: '1.25',
        letterSpacing: '0.0075rem',
      },
    },
    shadows: [
      'none',
      '0px 2px 1px -1px rgba(0, 0, 0, 0.2), 0px 1px 1px 0px rgba(0, 0, 0, 0.14), 0px 1px 3px 0px rgba(0, 0, 0, 0.12)',
      '0px 3px 1px -2px rgba(0, 0, 0, 0.2), 0px 2px 2px rgba(0, 0, 0, 0.14), 0px 1px 5px rgba(0, 0, 0, 0.12)',
      '0px 3px 3px -2px rgba(0, 0, 0, 0.2), 0px 3px 4px 0px rgba(0, 0, 0, 0.14), 0px 1px 8px 0px rgba(0, 0, 0, 0.12)',
      '0px 2px 4px -1px rgba(0, 0, 0, 0.2), 0px 4px 5px rgba(0, 0, 0, 0.14), 0px 1px 10px rgba(0, 0, 0, 0.12)',
      '0px 3px 5px -1px rgba(0, 0, 0, 0.2), 0px 5px 8px 0px rgba(0, 0, 0, 0.14), 0px 1px 14px 0px rgba(0, 0, 0, 0.12)',
      '0px 3px 5px -1px rgba(0, 0, 0, 0.2), 0px 6px 10px 0px rgba(0, 0, 0, 0.14), 0px 1px 18px 0px rgba(0, 0, 0, 0.12)',
      '0px 4px 5px -2px rgba(0, 0, 0, 0.2), 0px 7px 10px 1px rgba(0, 0, 0, 0.14), 0px 2px 16px 1px rgba(0, 0, 0, 0.12)',
      '0px 5px 5px -3px rgba(0, 0, 0, 0.2), 0px 8px 10px 1px rgba(0, 0, 0, 0.14), 0px 3px 14px 2px rgba(0, 0, 0, 0.12)',
      '0px 5px 6px -3px rgba(0, 0, 0, 0.2), 0px 9px 12px 1px rgba(0, 0, 0, 0.14), 0px 3px 16px 2px rgba(0, 0, 0, 0.12)',
      '0px 6px 6px -3px rgba(0, 0, 0, 0.2), 0px 10px 14px 1px rgba(0, 0, 0, 0.14), 0px 4px 18px 3px rgba(0, 0, 0, 0.12)',
      '0px 6px 7px -4px rgba(0, 0, 0, 0.2), 0px 11px 15px 1px rgba(0, 0, 0, 0.14), 0px 4px 20px 3px rgba(0, 0, 0, 0.12)',
      '0px 7px 8px -4px rgba(0, 0, 0, 0.2), 0px 12px 17px 2px rgba(0, 0, 0, 0.14), 0px 5px 22px 4px rgba(0, 0, 0, 0.12)',
      '0px 7px 8px -4px rgba(0, 0, 0, 0.2), 0px 13px 19px 2px rgba(0, 0, 0, 0.14), 0px 5px 24px 4px rgba(0, 0, 0, 0.12)',
      '0px 7px 9px -4px rgba(0, 0, 0, 0.2), 0px 14px 21px 2px rgba(0, 0, 0, 0.14), 0px 5px 26px 4px rgba(0, 0, 0, 0.12)',
      '0px 8px 9px -5px rgba(0, 0, 0, 0.2), 0px 15px 22px 2px rgba(0, 0, 0, 0.14), 0px 6px 28px 5px rgba(0, 0, 0, 0.12)',
      '0px 8px 10px -5px rgba(0, 0, 0, 0.2), 0px 16px 24px 2px rgba(0, 0, 0, 0.14), 0px 6px 30px 5px rgba(0, 0, 0, 0.12)',
      '0px 8px 11px -5px rgba(0, 0, 0, 0.2), 0px 17px 26px 2px rgba(0, 0, 0, 0.14), 0px 6px 32px 5px rgba(0, 0, 0, 0.12)',
      '0px 9px 11px -5px rgba(0, 0, 0, 0.2), 0px 18px 28px 2px rgba(0, 0, 0, 0.14), 0px 7px 34px 6px rgba(0, 0, 0, 0.12)',
      '0px 9px 12px -6px rgba(0, 0, 0, 0.2), 0px 19px 29px 2px rgba(0, 0, 0, 0.14), 0px 7px 36px 6px rgba(0, 0, 0, 0.12)',
      '0px 10px 13px -6px rgba(0, 0, 0, 0.2), 0px 20px 31px 3px rgba(0, 0, 0, 0.14), 0px 8px 38px 7px rgba(0, 0, 0, 0.12)',
      '0px 10px 13px -6px rgba(0, 0, 0, 0.2), 0px 21px 33px 3px rgba(0, 0, 0, 0.14), 0px 8px 40px 7px rgba(0, 0, 0, 0.12)',
      '0px 10px 14px -6px rgba(0, 0, 0, 0.2), 0px 22px 35px 3px rgba(0, 0, 0, 0.14), 0px 8px 42px 7px rgba(0, 0, 0, 0.12)',
      '0px 11px 14px -7px rgba(0, 0, 0, 0.2), 0px 23px 36px 3px rgba(0, 0, 0, 0.14), 0px 9px 44px 8px rgba(0, 0, 0, 0.12)',
      '0px 11px 15px -7px rgba(0, 0, 0, 0.2), 0px 24px 38px 3px rgba(0, 0, 0, 0.14), 0px 9px 46px 8px rgba(0, 0, 0, 0.12)',
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
            fontSize: '16px',
            fontWeight: 400,
            maxWidth: '100%',
          }),
        },
      },
      MuiButtonBase: {
        defaultProps: {
          disableRipple: true,
        },
        styleOverrides: {
          root: ({ theme }) => ({
            '&.MuiIconButton-root:not(:focus-visible):focus': {
              backgroundColor: 'transparent',
            },
            '&.MuiIconButton-root:focus-visible:focus': {
              backgroundColor: theme.palette.primary.focusVisible,
            },
            '&.MuiIconButton-root:focus-visible': {
              backgroundColor: theme.palette.primary.focusVisible,
            },
          }),
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

            '.MuiButton-startIcon': {
              height: 0,
              alignItems: 'center',
            },

            ...(ownerState.variant === 'contained' && {
              ...(ownerState.size === 'large' && {
                padding: '12px 24px',
              }),
              ...(ownerState.size === 'medium' && {
                padding: '11px 16px',
              }),
              ...(ownerState.size === 'small' && {
                padding: '8px 12px',
              }),
            }),

            ...(ownerState.variant === 'outlined' && {
              ...(ownerState.size === 'large' && {
                padding: '12px 22px',
              }),
              ...(ownerState.size === 'medium' && {
                padding: '11px 16px',
              }),
              ...(ownerState.size === 'small' && {
                padding: '8px 10px',
              }),
              borderColor: theme.palette.primary.main,
            }),

            ...(ownerState.variant === 'text' && {
              ...(ownerState.size === 'large' && {
                padding: '8px 11px',
              }),
              ...(ownerState.size === 'medium' && {
                padding: '6px 8px',
              }),
              ...(ownerState.size === 'small' && {
                padding: '4px 5px',
              }),
              borderColor: theme.palette.primary.main,
            }),

            ...(ownerState.size === 'large' && {
              fontSize: 15,
            }),
            ...(ownerState.size === 'medium' && {
              fontSize: 13,
            }),
            ...(ownerState.size === 'small' && {
              fontSize: 13,
            }),

            '&:hover': {
              borderWidth: '2px',
              ...(ownerState.variant === 'outlined' && {
                backgroundColor: theme.palette.action.focus,
              }),
              ...(ownerState.variant === 'text' && {
                backgroundColor: theme.palette.action.focus,
              }),
            },
            '&:disabled': {
              borderWidth: '2px',
              ...(ownerState.variant === 'contained' && {
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
            borderRadius: '128px',
          },
          grouped: ({ ownerState }) => ({
            '&:not(:last-of-type)': {
              ...(ownerState.variant === 'contained' && {
                borderRight: 0,
              }),
              marginLeft: '-2px',
            },
            '&:not(:first-of-type)': {
              marginLeft: '-2px',
            },
          }),
        },
      },
      MuiMenuItem: {
        styleOverrides: {
          root: ({ theme }) => ({
            ...theme.typography.menuText,
            // This should override any other nested typography (e.g. form labels)
            '.MuiTypography-root': {
              ...theme.typography.menuText,
            },
            '&.Mui-disabled': {
              opacity: 0.5,
            },
          }),
        },
      },
      MuiOutlinedInput: {
        styleOverrides: {
          root: ({ theme }) => ({
            ...theme.typography.inputText,
            '& fieldset': {
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
            '> *:first-child': {
              marginTop: 0,
            },
          }),
        },
      },
      MuiFormControl: {
        styleOverrides: {
          root: ({ theme }) => ({
            '.MuiInputLabel-root': {
              color: theme.palette.text.secondary,
              '&.Mui-focused': {
                color: theme.palette.text.primary,
              },
            },
          }),
        },
      },
      MuiFormHelperText: {
        styleOverrides: {
          root: {
            fontSize: '12px',
            fontWeight: 400,
          },
        },
      },
      MuiTabs: {
        styleOverrides: {
          root: {
            minHeight: '32px',
            '.MuiButtonBase-root': {
              fontSize: '16px',
              padding: '8px 12px',
            },
            '.MuiTabs-indicator': {
              height: '3px',
            },
            '.MuiTabScrollButton-root': {
              '&.Mui-disabled': {
                opacity: 0.1,
              },
            },
          },
        },
      },
      MuiDialogTitle: {
        defaultProps: {
          component: 'h5',
          variant: 'h5',
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
      MuiLink: {
        styleOverrides: {
          root: {
            color: tokens.text.accent1,
            textDecorationColor: tokens.text.accent1,
            '&:hover': {
              opacity: 0.8,
            },
            '&:active': {
              opacity: 0.6,
            },
            '&:focus-visible': {
              outline: `2px solid ${tokens.text.accent1}`,
              outlineOffset: '2px',
              borderRadius: '2px',
            },
          },
        },
      },
      MuiAccordion: {
        styleOverrides: {
          root: ({ theme }) => ({
            borderWidth: '1px',
            borderStyle: 'solid',
            borderColor: 'rgba(44, 50, 62, 0.25)', // TODO move into pallet =#2C323E 25%
            borderRadius: theme.spacing(1),
            backgroundColor: theme.palette.action.hover,
            boxShadow: 'none',
            '&:before': {
              height: 0,
            },
            marginTop: '8px',
            '&.Mui-expanded': {
              marginTop: '8px',
            },
          }),
        },
      },
      MuiAccordionSummary: {
        styleOverrides: {
          root: () => ({
            '&.Mui-disabled': {
              opacity: '0.5 !important',
            },
          }),
        },
      },
      MuiCard: {
        variants: [
          {
            props: { variant: 'grey' },
            style: ({ theme }) => ({
              border: `${theme.palette.dividers?.divider} 1px solid`,
              '.MuiCardHeader-root': {
                backgroundColor: theme.palette.surfaces?.low,
                borderBottomColor: theme.palette.dividers?.divider,
                borderBottomWidth: 1,
                borderBottomStyle: 'solid',
                padding: `${theme.spacing(1.5)} ${theme.spacing(2)}`,
                '.MuiTypography-root': {
                  ...theme.typography.sectionHeading,
                },
                '#database-icon, #network-node-icon': {
                  path: { fill: theme.palette.text.primary },
                },
              },
              '.MuiCardContent': {
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
            '.MuiPickersCalendarHeader-root': {
              marginTop: theme.spacing(2),
              marginBottom: theme.spacing(2),
            },
            '.MuiPickersCalendarHeader-label': {
              fontSize: theme.typography.inputText.fontSize,
              fontWeight: theme.typography.inputText.fontWeight,
            },
            '.MuiDayCalendar-weekDayLabel': {
              fontSize: theme.typography.helperText.fontSize,
              fontWeight: theme.typography.helperText.fontWeight,
              color: theme.palette.text.disabled,
            },
            '.MuiPickersDay-root': {
              fontSize: theme.typography.body2.fontSize,
              fontWeight: theme.typography.body2.fontWeight,
            },
          }),
        },
      },
      MuiMultiSectionDigitalClock: {
        styleOverrides: {
          root: () => ({
            '.MuiMultiSectionDigitalClockSection-root': {
              '&::after': {
                height: 'calc(100% - 40px - 2px)',
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
            borderRadius: '4px',
            borderWidth: '1px',
            borderStyle: 'solid',
            borderColor: theme.palette.mode === 'light' ? 'rgba(0, 0, 0, 0.059)' : 'transparent',
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
          filled: ({ theme, ownerState: { color } }) => {
            if (!color || color === 'default') {
              return {
                backgroundColor: theme.palette.action.focus,
                color: theme.palette.text.primary,
              };
            }
            const paletteColor = theme.palette[color];
            return {
              backgroundColor: paletteColor.surface,
              color: paletteColor.contrastText,
            };
          },
          outlined: ({ theme, ownerState: { color } }) => {
            const base = { backgroundColor: 'transparent' };
            if (!color || color === 'default') {
              return {
                ...base,
                color: theme.palette.text.primary,
                borderColor: theme.palette.text.primary,
              };
            }
            const tone =
              color === 'warning' ? theme.palette.warning.main : theme.palette[color].light;
            return { ...base, color: tone, borderColor: tone };
          },
          label: ({ theme, ownerState }) => ({
            ...(ownerState.size === 'small'
              ? theme.typography.helperText
              : theme.typography.inputText),
            paddingLeft: ownerState.size === 'small' ? 5 : 7,
            paddingRight: ownerState.size === 'small' ? 5 : 7,
          }),
          avatar: ({ ownerState }) => ({
            width: 24,
            height: 24,
            marginLeft: ownerState.size === 'small' ? 0 : 4,
            marginRight: ownerState.size === 'small' ? -1 : -3,
          }),
          deleteIcon: ({ theme, ownerState }) => {
            const isSmall = ownerState.size === 'small';
            const iconSize = isSmall ? 16 : 20;
            return {
              display: 'inline-flex',
              alignItems: 'center',
              justifyContent: 'center',
              padding: isSmall ? 4 : 6,
              marginLeft: isSmall ? -5 : -7,
              marginRight: 0,
              cursor: 'pointer',
              color: theme.palette.text.disabled,
              '& > svg': {
                width: iconSize,
                height: iconSize,
                fontSize: iconSize,
              },
              '&:hover': {
                color: theme.palette.text.primary,
              },
            };
          },
        },
      },
      MuiTableHead: {
        styleOverrides: {
          root: () => ({
            '.MuiBadge-overlapCircular': {
              backgroundColor: 'transparent',
            },
          }),
        },
      },

      MuiTableBody: {
        styleOverrides: {
          root: ({ theme }) => ({
            '#empty-state-icon': {
              path: {
                // complex selector we need in order to provide dark theme style for this icon, instead of having a separate one
                '&:not(:nth-child(n+8)), &:last-child': {
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
