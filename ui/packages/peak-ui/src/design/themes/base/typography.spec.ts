import { createTheme, type PaletteMode, type Theme } from '@mui/material/styles';
import { describe, expect, it } from 'vitest';
import { getThemeOptions } from '../../utils';

const THEMES = ['base', 'pmm', 'sep'] as const;
const MODES: PaletteMode[] = ['light', 'dark'];

// Line px per width range (below sm, from sm), as in the Figma Tokens: Typography page.
const FIGMA_LINES: Record<string, number[]> = {
  h1: [34, 51],
  h2: [31, 43],
  h3: [28, 35],
  h4: [24, 30],
  h5: [23, 26],
  h6: [23, 24],
  subHead1: [21],
  subHead2: [20],
  body1: [22],
  body2: [21],
  caption: [18],
  button: [16],
  menuText: [18],
  overline: [12],
  inputText: [24],
  inputLabel: [12],
  helperText: [15],
};

// Off-pixel line on main that peak-ui #70 restyles to Figma's 16/20px.
// CONTRIBUTING and Foundations/Typography name the same exception; drop it from all three together.
const PENDING_RESTYLE: Record<string, number> = { sectionHeading: 17.5 };

type Style = Record<string, unknown>;

interface Line {
  variant: string;
  range: string;
  fontSize: unknown;
  lineHeight: unknown;
}

const isStyle = (value: unknown): value is Style => typeof value === 'object' && value !== null;

const toPx = (fontSize: unknown, htmlFontSize: number): number => {
  if (typeof fontSize === 'number') return fontSize;
  const match = /^(\d*\.?\d+)(px|rem)$/.exec(String(fontSize));
  if (!match) throw new Error(`Unsupported font size: ${String(fontSize)}`);
  return Number(match[1]) * (match[2] === 'rem' ? htmlFontSize : 1);
};

// The font size and line height a variant renders with in each width range it styles.
const renderedLines = (theme: Theme, variant: string, style: Style): Line[] => {
  const queries = Object.keys(style).filter((key) => key.startsWith('@media'));
  const coversAllWidths = theme.breakpoints.keys.some(
    (key) =>
      queries.includes(theme.breakpoints.down(key)) && queries.includes(theme.breakpoints.up(key))
  );
  const ranges = queries.map((query) => {
    const block = style[query] as Style;
    return {
      variant,
      range: query,
      fontSize: block.fontSize ?? style.fontSize,
      lineHeight: block.lineHeight ?? style.lineHeight,
    };
  });
  const base = { variant, range: 'base', fontSize: style.fontSize, lineHeight: style.lineHeight };
  return coversAllWidths ? ranges : [base, ...ranges];
};

const allLines = (theme: Theme): Line[] =>
  Object.entries(theme.typography).flatMap(([variant, style]) =>
    isStyle(style) && variant !== 'inherit' && ('fontSize' in style || 'lineHeight' in style)
      ? renderedLines(theme, variant, style)
      : []
  );

const linePx = (theme: Theme, { fontSize, lineHeight }: Line) =>
  toPx(fontSize, theme.typography.htmlFontSize) * Number(lineHeight);

const offPixel = (theme: Theme, line: Line): string | null => {
  const { variant, range, lineHeight } = line;
  const unitless = typeof lineHeight === 'number' || /^\d*\.?\d+$/.test(String(lineHeight ?? ''));
  if (!unitless) return `${variant} (${range}): line height ${String(lineHeight)} is not a ratio`;
  const px = linePx(theme, line);
  const above = px - Math.round(px);
  if (above >= 0 && above <= 0.005) return null;
  return PENDING_RESTYLE[variant] === px ? null : `${variant} (${range}): ${px}px`;
};

describe.each(THEMES)('%s theme typography', (themeName) => {
  describe.each(MODES)('%s mode', (mode) => {
    const theme = createTheme(getThemeOptions(themeName)(mode));

    it('lands every variant on whole pixels at every breakpoint', () => {
      const lines = allLines(theme);
      expect(lines.length).toBeGreaterThan(0);
      expect(lines.map((line) => offPixel(theme, line)).filter(Boolean)).toEqual([]);
    });

    it('pins only lines that are still off-pixel', () => {
      const pinned = allLines(theme).filter(({ variant }) => variant in PENDING_RESTYLE);
      expect(Object.fromEntries(pinned.map((line) => [line.variant, linePx(theme, line)]))).toEqual(
        PENDING_RESTYLE
      );
    });

    it.each(Object.entries(FIGMA_LINES))('gives %s the Figma line heights', (variant, expected) => {
      const style = theme.typography[variant as keyof Theme['typography']];
      expect(isStyle(style)).toBe(true);
      const lines = renderedLines(theme, variant, style as Style).map((line) =>
        Math.round(linePx(theme, line))
      );
      expect(lines).toEqual(expected);
    });
  });
});
