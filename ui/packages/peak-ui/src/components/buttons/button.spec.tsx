import { describe, expect, it } from 'vitest';
import { render } from '@testing-library/react';
import { createTheme, ThemeProvider } from '@mui/material/styles';
import type { Theme, PaletteMode } from '@mui/material/styles';
import type { ButtonProps } from '@mui/material/Button';
import Button from '@mui/material/Button';
import { getThemeOptions } from '../../design';

const THEMES = ['base', 'pmm', 'sep'] as const;
const MODES: PaletteMode[] = ['light', 'dark'];
const VARIANTS = ['contained', 'outlined', 'text'] as const;
const SIZES = ['large', 'medium', 'small'] as const;

const buildTheme = (name: string, mode: PaletteMode) => createTheme(getThemeOptions(name)(mode));

type RootStyleFn = (props: { ownerState: ButtonProps; theme: Theme }) => Record<string, unknown>;

const rootStyle = (theme: Theme, ownerState: ButtonProps): Record<string, unknown> => {
  const root = theme.components?.MuiButton?.styleOverrides?.root;
  expect(typeof root).toBe('function');
  return (root as RootStyleFn)({ ownerState, theme });
};

const ELEVATION_2 =
  '0px 0px 1px 0px rgba(0,0,0,0.24), 0px 2px 2px 0px rgba(0,0,0,0.2), 0px 2px 4px 0px rgba(0,0,0,0.24)';
const ELEVATION_4 =
  '0px 0px 1px 0px rgba(0,0,0,0.2), 0px 4px 2px 0px rgba(0,0,0,0.16), 0px 4px 8px 0px rgba(0,0,0,0.22)';

const PADDING: Record<(typeof VARIANTS)[number], Record<(typeof SIZES)[number], string>> = {
  contained: { large: '4px 24px', medium: '5px 18px', small: '4px 9px' },
  outlined: { large: '4px 24px', medium: '5px 18px', small: '4px 9px' },
  text: { large: '0px 8px', medium: '0px 7px', small: '0px 4px' },
};

const MIN_HEIGHT: Record<(typeof SIZES)[number], number> = { large: 40, medium: 32, small: 24 };
const FONT_SIZE: Record<(typeof SIZES)[number], number> = { large: 15, medium: 13, small: 13 };

describe.each(THEMES)('MuiButton design contract — %s theme', (themeName) => {
  describe.each(MODES)('%s mode', (mode) => {
    const theme = buildTheme(themeName, mode);

    it.each(VARIANTS.flatMap((v) => SIZES.map((s) => [v, s] as const)))(
      'styles %s / %s per the design kit',
      (variant, size) => {
        const style = rootStyle(theme, { variant, size });

        expect(style.borderRadius).toBe(theme.shape.borderRadiusFull);
        expect(style.padding).toBe(PADDING[variant][size]);
        expect(style.minHeight).toBe(MIN_HEIGHT[size]);
        expect(style.fontSize).toBe(FONT_SIZE[size]);
        expect(style.lineHeight).toBe(1.063);

        const hover = style['&:hover'] as Record<string, unknown>;
        const disabled = style['&.Mui-disabled'] as Record<string, unknown>;

        if (variant === 'contained') {
          expect(style.boxShadow).toBe(ELEVATION_2);
          expect(style.border).toBe('2px solid transparent');
          expect(hover.backgroundColor).toBe(theme.palette.primary.dark);
          expect(hover.boxShadow).toBe(ELEVATION_4);
          expect(disabled.backgroundColor).toBe(theme.palette.action.disabled);
          expect(disabled.boxShadow).toBe('none');
        }

        if (variant === 'outlined') {
          expect(style.borderWidth).toBe(2);
          expect(style.borderColor).toBe(theme.palette.primary.main);
          expect(hover.backgroundColor).toBe(theme.palette.primary.hover);
          expect(hover.borderColor).toBe(theme.palette.primary.light);
          expect(hover.color).toBe(theme.palette.primary.light);
          expect(hover.borderWidth).toBe(2);
        }

        if (variant === 'text') {
          expect(style.border).toBe('none');
          expect(hover.backgroundColor).toBe(theme.palette.primary.hover);
          expect(hover.color).toBe(theme.palette.primary.light);
        }

        expect(disabled.color).toBe(theme.palette.text.disabled);
      }
    );

    it.each(SIZES)('sizes and positions icons per the design kit at size %s', (size) => {
      const iconSlot = 12;
      const iconSize = size === 'large' ? 24 : size === 'medium' ? 20 : 16;
      const edgePull = iconSize - iconSlot;

      for (const variant of VARIANTS) {
        const style = rootStyle(theme, { variant, size });
        const iconChild = (
          style['.MuiButton-startIcon, .MuiButton-endIcon'] as Record<
            string,
            Record<string, unknown>
          >
        )['& > *:nth-of-type(1)'];
        const startIcon = style['.MuiButton-startIcon'] as Record<string, unknown>;
        const endIcon = style['.MuiButton-endIcon'] as Record<string, unknown>;

        expect(iconChild.fontSize).toBe(iconSize);
        expect(iconChild.width).toBe(iconSize);
        expect(iconChild.height).toBe(iconSize);

        if (variant === 'text') {
          expect(startIcon.marginLeft).toBe(0);
          expect(startIcon.marginRight).toBe(4);
          expect(endIcon.marginLeft).toBe(4);
          expect(endIcon.marginRight).toBe(0);
        } else {
          expect(startIcon.marginLeft).toBe(-edgePull);
          expect(startIcon.marginRight).toBe(4);
          expect(endIcon.marginLeft).toBe(4);
          expect(endIcon.marginRight).toBe(-edgePull);
        }
      }
    });

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
