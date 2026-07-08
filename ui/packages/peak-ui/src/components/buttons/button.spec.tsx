import { describe, expect, it } from 'vitest';
import { render } from '@testing-library/react';
import { createTheme, ThemeProvider } from '@mui/material/styles';
import type { Theme, PaletteMode } from '@mui/material';
import type { ButtonProps } from '@mui/material/Button';
import Button from '@mui/material/Button';
import { getThemeOptions } from '../../design';

const THEMES = ['base', 'pmm', 'sep'] as const;
const MODES: PaletteMode[] = ['light', 'dark'];
const VARIANTS = ['contained', 'outlined', 'text'] as const;
const SIZES = ['large', 'medium', 'small'] as const;

const buildTheme = (name: string, mode: PaletteMode) => createTheme(getThemeOptions(name)(mode));

type RootStyleFn = (props: { ownerState: ButtonProps; theme: Theme }) => Record<string, never>;

const rootStyle = (theme: Theme, ownerState: ButtonProps): Record<string, never> => {
  const root = theme.components?.MuiButton?.styleOverrides?.root;
  expect(typeof root).toBe('function');
  return (root as RootStyleFn)({ ownerState, theme });
};

// The design-kit contract: pill shape, 2px border, per-variant/size padding,
// 15/13px labels, action-token state colors. A theme change that breaks any of
// these silently changes every button in every consuming product.
const PADDING: Record<(typeof VARIANTS)[number], Record<(typeof SIZES)[number], string>> = {
  contained: { large: '12px 24px', medium: '11px 16px', small: '8px 12px' },
  outlined: { large: '12px 22px', medium: '11px 16px', small: '8px 10px' },
  text: { large: '8px 11px', medium: '6px 8px', small: '4px 5px' },
};

const FONT_SIZE: Record<(typeof SIZES)[number], number> = { large: 15, medium: 13, small: 13 };

describe.each(THEMES)('MuiButton design contract — %s theme', (themeName) => {
  describe.each(MODES)('%s mode', (mode) => {
    const theme = buildTheme(themeName, mode);

    it.each(VARIANTS.flatMap((v) => SIZES.map((s) => [v, s] as const)))(
      'styles %s / %s per the design kit',
      (variant, size) => {
        const style = rootStyle(theme, { variant, size });

        expect(style.borderRadius).toBe(theme.shape.borderRadiusFull);
        expect(style.borderWidth).toBe(2);
        expect(style.padding).toBe(PADDING[variant][size]);
        expect(style.fontSize).toBe(FONT_SIZE[size]);

        const hover = style['&:hover'] as Record<string, unknown>;
        expect(hover.borderWidth).toBe('2px');
        if (variant !== 'contained') {
          expect(style.borderColor).toBe(theme.palette.primary.main);
          expect(hover.backgroundColor).toBe(theme.palette.action.focus);
        }

        const disabled = style['&:disabled'] as Record<string, unknown>;
        expect(disabled.color).toBe(theme.palette.text.disabled);
        if (variant === 'contained') {
          expect(disabled.backgroundColor).toBe(theme.palette.action.disabled);
        }
      }
    );

    it('uses the design-kit button typography', () => {
      expect(theme.typography.button.fontWeight).toBe(600);
      expect(theme.typography.button.textTransform).toBe('none');
      expect(theme.typography.button.fontSize).toBe('0.9375rem');
    });

    it('disables ripples via defaultProps', () => {
      expect(theme.components?.MuiButtonBase?.defaultProps?.disableRipple).toBe(true);
      expect(theme.components?.MuiButton?.defaultProps?.disableTouchRipple).toBe(true);
    });

    it.each(VARIANTS.flatMap((v) => SIZES.map((s) => [v, s] as const)))(
      'renders %s / %s as a native themed button',
      (variant, size) => {
        const { getByRole, unmount } = render(
          <ThemeProvider theme={theme}>
            <Button variant={variant} size={size}>
              Label
            </Button>
          </ThemeProvider>
        );
        const button = getByRole('button', { name: 'Label' });
        expect(button.tagName).toBe('BUTTON');
        expect(button).toHaveClass(`MuiButton-${variant}`);
        expect(button).toHaveClass(`MuiButton-size${size[0].toUpperCase()}${size.slice(1)}`);
        unmount();
      }
    );
  });
});

describe('SEP contained semantic colors', () => {
  it.each(MODES)('keep readable labels in %s mode', (mode) => {
    const theme = buildTheme('sep', mode);
    const variants = theme.components?.MuiButton?.variants ?? [];
    const containedColors = variants
      .map((v) => v.props as ButtonProps)
      .filter((p) => p.variant === 'contained')
      .map((p) => p.color);

    expect(containedColors).toEqual(
      expect.arrayContaining(['success', 'error', 'warning', 'info'])
    );
    variants.forEach((v) => {
      expect((v.style as Record<string, unknown>).color).toBeTruthy();
    });
  });

  it('base and pmm themes add no button variants', () => {
    for (const name of ['base', 'pmm']) {
      expect(buildTheme(name, 'light').components?.MuiButton?.variants ?? []).toHaveLength(0);
    }
  });
});
