import { Theme, createTheme } from '@mui/material';
import { adreColors, headingFont } from './adreAnswerSx';

const INPUT_BG = adreColors.card;
const HOVER_BORDER = 'rgba(255,255,255,0.16)';

const alertTone = (color: string, rgb: string) => ({
  color,
  backgroundColor: `rgba(${rgb},0.1)`,
  border: `1px solid rgba(${rgb},0.25)`,
  '& .MuiAlert-icon': { color },
});

/** The Assistant's dark look for MUI components, whatever PMM's own theme is. */
export const adreTheme = (outerTheme: Theme): Theme => {
  // Rendered without an app theme around it (a bare test render), there is nothing to extend.
  const outer = outerTheme?.palette ? outerTheme : createTheme();
  return {
    ...outer,
    palette: {
      ...outer.palette,
      mode: 'dark',
      primary: {
        ...outer.palette.primary,
        main: adreColors.accent,
        light: adreColors.accentText,
        dark: '#9893F0',
        contrastText: adreColors.panel,
      },
      background: { default: adreColors.page, paper: adreColors.panel },
      surfaces: {
        ...outer.palette.surfaces,
        low: adreColors.page,
        high: adreColors.panel,
      },
      divider: adreColors.border,
      dividers: {
        ...outer.palette.dividers,
        divider: adreColors.border,
        dividerStrong: HOVER_BORDER,
      },
      text: {
        ...outer.palette.text,
        primary: adreColors.text,
        secondary: adreColors.secondary,
        disabled: adreColors.muted,
      },
      action: {
        ...outer.palette.action,
        active: adreColors.secondary,
        hover: 'rgba(255,255,255,0.05)',
        selected: 'rgba(182,178,255,0.12)',
        focus: 'rgba(182,178,255,0.16)',
        disabled: 'rgba(255,255,255,0.3)',
        disabledBackground: 'rgba(255,255,255,0.08)',
      },
    },
    typography: {
      ...outer.typography,
      h4: { ...outer.typography.h4, fontFamily: headingFont, fontWeight: 600 },
      subtitle1: { ...outer.typography.subtitle1, fontFamily: headingFont },
      h5: { ...outer.typography.h5, fontFamily: headingFont, fontWeight: 600 },
      h6: { ...outer.typography.h6, fontFamily: headingFont, fontWeight: 600 },
    },
    components: {
      ...outer.components,
      MuiPaper: {
        ...outer.components?.MuiPaper,
        // MUI lightens raised dark surfaces (menus, dialogs) with a white overlay.
        styleOverrides: { root: { backgroundImage: 'none' } },
      },
      MuiCard: {
        ...outer.components?.MuiCard,
        styleOverrides: {
          root: {
            backgroundColor: adreColors.panel,
            border: `1px solid ${adreColors.divider}`,
            borderRadius: 16,
            boxShadow: 'none',
          },
        },
      },
      MuiButton: {
        ...outer.components?.MuiButton,
        styleOverrides: {
          root: {
            borderRadius: 999,
            textTransform: 'none',
            fontWeight: 500,
            boxShadow: 'none',
            '&:hover': { boxShadow: 'none' },
          },
          outlined: { borderColor: HOVER_BORDER, color: adreColors.text },
          text: { color: adreColors.accentText },
        },
      },
      MuiOutlinedInput: {
        ...outer.components?.MuiOutlinedInput,
        styleOverrides: {
          root: {
            backgroundColor: INPUT_BG,
            borderRadius: 10,
            color: adreColors.text,
            '& .MuiOutlinedInput-notchedOutline': {
              borderColor: adreColors.border,
            },
            '&:hover .MuiOutlinedInput-notchedOutline': {
              borderColor: HOVER_BORDER,
            },
            '&.Mui-focused .MuiOutlinedInput-notchedOutline': {
              borderColor: adreColors.accent,
            },
          },
        },
      },
      MuiInputLabel: {
        ...outer.components?.MuiInputLabel,
        styleOverrides: {
          root: {
            color: adreColors.secondary,
            '&.Mui-focused': { color: adreColors.accentText },
          },
        },
      },
      MuiFormHelperText: {
        ...outer.components?.MuiFormHelperText,
        styleOverrides: { root: { color: adreColors.muted } },
      },
      MuiTabs: {
        ...outer.components?.MuiTabs,
        styleOverrides: {
          root: { borderBottom: `1px solid ${adreColors.divider}` },
          indicator: { backgroundColor: adreColors.accent },
        },
      },
      MuiTab: {
        ...outer.components?.MuiTab,
        styleOverrides: {
          root: {
            textTransform: 'none',
            fontWeight: 500,
            color: adreColors.secondary,
            '&.Mui-selected': { color: adreColors.text },
          },
        },
      },
      MuiTableCell: {
        ...outer.components?.MuiTableCell,
        styleOverrides: {
          root: { borderColor: adreColors.divider, color: adreColors.body },
          head: { color: adreColors.muted, fontSize: 12, fontWeight: 500 },
        },
      },
      MuiAlert: {
        ...outer.components?.MuiAlert,
        styleOverrides: {
          root: { borderRadius: 12 },
          standardWarning: alertTone('#FCD34D', '252,211,77'),
          standardError: alertTone('#FCA5A5', '248,113,113'),
          standardSuccess: alertTone('#86EFAC', '134,239,172'),
          standardInfo: alertTone(adreColors.accentText, '182,178,255'),
          outlinedWarning: alertTone('#FCD34D', '252,211,77'),
          outlinedError: alertTone('#FCA5A5', '248,113,113'),
          outlinedSuccess: alertTone('#86EFAC', '134,239,172'),
          outlinedInfo: alertTone(adreColors.accentText, '182,178,255'),
        },
      },
      MuiChip: {
        ...outer.components?.MuiChip,
        styleOverrides: { root: { borderRadius: 999 } },
      },
      MuiDialog: {
        ...outer.components?.MuiDialog,
        styleOverrides: {
          paper: {
            backgroundColor: adreColors.panel,
            border: `1px solid ${adreColors.border}`,
            borderRadius: 16,
          },
        },
      },
      MuiPopover: {
        ...outer.components?.MuiPopover,
        styleOverrides: {
          paper: {
            backgroundColor: adreColors.card,
            border: `1px solid ${adreColors.border}`,
          },
        },
      },
      MuiDivider: {
        ...outer.components?.MuiDivider,
        styleOverrides: { root: { borderColor: adreColors.divider } },
      },
    },
  };
};
