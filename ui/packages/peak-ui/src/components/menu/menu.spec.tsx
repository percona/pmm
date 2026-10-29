import { describe, expect, it } from 'vitest';
import { createTheme } from '@mui/material/styles';
import type { PaletteMode, Theme } from '@mui/material/styles';
import { getThemeOptions } from '../../design';

const THEMES = ['base', 'pmm', 'sep'] as const;
const MODES: PaletteMode[] = ['light', 'dark'];

const ELEVATION_8 =
  '0px 0px 1px 0px rgba(0,0,0,0.16), 0px 8px 4px 0px rgba(0,0,0,0.12), 0px 8px 16px 0px rgba(0,0,0,0.2)';

type StyleFn = (props: {
  ownerState: Record<string, unknown>;
  theme: Theme;
}) => Record<string, unknown>;

const slotStyle = (
  theme: Theme,
  component: 'MuiPopover' | 'MuiMenuItem' | 'MuiListItemButton' | 'MuiListSubheader',
  slot: string,
  ownerState: Record<string, unknown> = {}
) => {
  const style = (theme.components?.[component]?.styleOverrides as Record<string, unknown>)?.[slot];
  expect(typeof style).toBe('function');
  return (style as StyleFn)({ ownerState, theme });
};

describe.each(THEMES)('Menu design contract — %s theme', (themeName) => {
  describe.each(MODES)('%s mode', (mode) => {
    const theme = createTheme(getThemeOptions(themeName)(mode));

    it('puts every popover on the Figma Menu surface', () => {
      expect(slotStyle(theme, 'MuiPopover', 'paper')).toEqual({
        borderRadius: theme.shape.borderRadiusMd,
        border: `1px solid ${theme.palette.divider}`,
        boxShadow: ELEVATION_8,
        backgroundImage: 'none',
        scrollbarWidth: 'thin',
        scrollbarColor: `${theme.palette.divider} ${theme.palette.action.hover}`,
        '&:has(.MuiListSubheader-sticky)': { scrollPaddingTop: 32 },
      });
    });

    it('keeps a menu at least as wide as the Figma Menu', () => {
      expect(theme.components?.MuiMenu?.styleOverrides?.paper).toEqual({ minWidth: 200 });
    });

    it('sets 40px rows from the sm breakpoint and keeps MUI body1 labels', () => {
      const row = slotStyle(theme, 'MuiMenuItem', 'root', { dense: false });
      const dense = slotStyle(theme, 'MuiMenuItem', 'root', { dense: true });

      expect(row[theme.breakpoints.up('sm')]).toEqual({ minHeight: 40 });
      expect(row).not.toHaveProperty('fontFamily');
      expect(Object.keys(dense)).not.toContain(theme.breakpoints.up('sm'));
    });

    it.each(['MuiMenuItem', 'MuiListItemButton'] as const)(
      'tints %s hover, focus and selected states with the theme’s primary tokens',
      (component) => {
        const { primary } = theme.palette;

        expect(slotStyle(theme, component, 'root', { dense: false })).toMatchObject({
          '&:hover': { backgroundColor: primary.hover },
          '&.Mui-focusVisible': { backgroundColor: primary.focus },
          '&.Mui-selected': { backgroundColor: primary.selected },
        });
      }
    );

    it.each(['MuiMenuItem', 'MuiListItemButton'] as const)(
      'keeps a selected %s on the selected token while hovered or keyboard-focused',
      (component) => {
        const { primary } = theme.palette;

        expect(slotStyle(theme, component, 'root', { dense: false })).toMatchObject({
          '&.Mui-selected': {
            backgroundColor: primary.selected,
            '&:hover, &.Mui-focusVisible': { backgroundColor: primary.selected },
          },
        });
      }
    );

    it('heads a run of rows with the overline', () => {
      expect(slotStyle(theme, 'MuiListSubheader', 'root')).toMatchObject({
        fontFamily: theme.typography.overline.fontFamily,
        fontSize: theme.typography.overline.fontSize,
        lineHeight: theme.typography.overline.lineHeight,
        minHeight: 32,
        paddingTop: theme.spacing(1),
        paddingBottom: theme.spacing(1),
      });
    });
  });
});

describe.each(MODES)('Menu row states across themes — %s mode', (mode) => {
  it('hovers neutral in Base and purple in PMM', () => {
    const hover = (name: 'base' | 'pmm') =>
      createTheme(getThemeOptions(name)(mode)).palette.primary.hover;

    expect(hover('base')).not.toBe(hover('pmm'));
  });
});
