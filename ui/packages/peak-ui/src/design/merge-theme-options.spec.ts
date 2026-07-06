import type { ThemeOptions } from '@mui/material';
import { describe, expect, it } from 'vitest';
import baseThemeOptions from './themes/base/BaseTheme';
import { mergeThemeOptions } from './merge-theme-options';

const mockTheme = {
  palette: {
    mode: 'light',
    divider: '#ccc',
    background: { default: '#fafafa', paper: '#fff' },
  },
} as const;

describe('mergeThemeOptions', () => {
  it('composes CssBaseline object overrides with child function overrides', () => {
    const merged = mergeThemeOptions(baseThemeOptions('light'), {
      components: {
        MuiCssBaseline: {
          styleOverrides: () => ({
            body: { backgroundColor: 'blue' },
          }),
        },
      },
    });

    const styleOverrides = merged.components?.MuiCssBaseline?.styleOverrides;
    expect(typeof styleOverrides).toBe('function');

    const resolved = (styleOverrides as (theme: typeof mockTheme) => Record<string, unknown>)(
      mockTheme
    );

    expect(resolved['code:not(pre code)']).toBeDefined();
    expect(resolved.body).toMatchObject({ backgroundColor: 'blue' });
  });

  it('composes slot-level style override functions', () => {
    const base: ThemeOptions = {
      components: {
        MuiLinearProgress: {
          styleOverrides: {
            root: () => ({ height: 4 }),
            bar: () => ({ borderRadius: 2 }),
          },
        },
      },
    };

    const override: ThemeOptions = {
      components: {
        MuiLinearProgress: {
          styleOverrides: {
            root: () => ({ height: 10 }),
          },
        },
      },
    };

    const merged = mergeThemeOptions(base, override);
    const root = merged.components?.MuiLinearProgress?.styleOverrides?.root;

    expect(typeof root).toBe('function');
    expect((root as () => Record<string, unknown>)()).toEqual({ height: 10 });

    const bar = merged.components?.MuiLinearProgress?.styleOverrides?.bar;
    expect(typeof bar).toBe('function');
    expect((bar as () => Record<string, unknown>)()).toEqual({ borderRadius: 2 });
  });

  it('lets child overrides win on conflicting keys while preserving base slots', () => {
    const merged = mergeThemeOptions(baseThemeOptions('light'), {
      components: {
        MuiCssBaseline: {
          styleOverrides: {
            'code:not(pre code)': {
              color: 'hotpink',
            },
          },
        },
      },
    });

    const inlineCode = merged.components?.MuiCssBaseline?.styleOverrides?.['code:not(pre code)'];

    expect(inlineCode).toMatchObject({ color: 'hotpink' });
    expect(inlineCode).toHaveProperty('fontFamily');
    expect(inlineCode).toHaveProperty('backgroundColor');
  });

  it('composes a base object slot with a child function slot', () => {
    const base: ThemeOptions = {
      components: {
        MuiButton: {
          styleOverrides: {
            root: { padding: 8, color: 'black' },
          },
        },
      },
    };

    const override: ThemeOptions = {
      components: {
        MuiButton: {
          styleOverrides: {
            root: () => ({ color: 'red' }),
          },
        },
      },
    };

    const merged = mergeThemeOptions(base, override);
    const root = merged.components?.MuiButton?.styleOverrides?.root;

    expect(typeof root).toBe('function');
    expect((root as () => Record<string, unknown>)()).toEqual({ padding: 8, color: 'red' });
  });

  it('composes a base function slot with a child object slot', () => {
    const base: ThemeOptions = {
      components: {
        MuiButton: {
          styleOverrides: {
            root: () => ({ padding: 8, color: 'black' }),
          },
        },
      },
    };

    const override: ThemeOptions = {
      components: {
        MuiButton: {
          styleOverrides: {
            root: { color: 'red' },
          },
        },
      },
    };

    const merged = mergeThemeOptions(base, override);
    const root = merged.components?.MuiButton?.styleOverrides?.root;

    expect(typeof root).toBe('function');
    expect((root as () => Record<string, unknown>)()).toEqual({ padding: 8, color: 'red' });
  });

  it('concatenates variants base-first so base variants survive and child cascades last', () => {
    const base: ThemeOptions = {
      components: {
        MuiCard: {
          variants: [{ props: { variant: 'grey' }, style: { border: '1px solid grey' } }],
        },
      },
    };

    const override: ThemeOptions = {
      components: {
        MuiCard: {
          variants: [{ props: { variant: 'brand' }, style: { border: '1px solid blue' } }],
        },
      },
    };

    const merged = mergeThemeOptions(base, override);
    const variants = merged.components?.MuiCard?.variants;

    expect(variants).toHaveLength(2);
    expect(variants?.[0]).toMatchObject({ props: { variant: 'grey' } });
    expect(variants?.[1]).toMatchObject({ props: { variant: 'brand' } });
  });

  it('keeps base variants when child only overrides styleOverrides on the same component', () => {
    const base: ThemeOptions = {
      components: {
        MuiCard: {
          variants: [{ props: { variant: 'grey' }, style: { border: '1px solid grey' } }],
          styleOverrides: { root: { borderRadius: 4 } },
        },
      },
    };

    const override: ThemeOptions = {
      components: {
        MuiCard: {
          styleOverrides: { root: { borderRadius: 8 } },
        },
      },
    };

    const merged = mergeThemeOptions(base, override);

    expect(merged.components?.MuiCard?.variants).toHaveLength(1);
    expect(merged.components?.MuiCard?.styleOverrides?.root).toMatchObject({ borderRadius: 8 });
  });

  it('preserves child-only components and defaultProps untouched', () => {
    const base: ThemeOptions = {
      components: {
        MuiButton: { defaultProps: { disableRipple: true } },
      },
    };

    const override: ThemeOptions = {
      components: {
        MuiChip: { defaultProps: { size: 'small' } },
      },
    };

    const merged = mergeThemeOptions(base, override);

    expect(merged.components?.MuiButton?.defaultProps).toMatchObject({ disableRipple: true });
    expect(merged.components?.MuiChip?.defaultProps).toMatchObject({ size: 'small' });
  });
});
