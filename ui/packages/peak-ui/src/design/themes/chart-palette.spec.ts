import {
  createTheme,
  getContrastRatio,
  type CSSObject,
  type PaletteMode,
  type Theme,
} from '@mui/material/styles';
import { describe, expect, it } from 'vitest';
import { CHART_SERIES_COLORS, getReadableTextColor } from '../chart-colors';
import { getThemeOptions } from '../utils';
import { semanticTokensDark, semanticTokensLight } from './base';
import { sepTokensDark, sepTokensLight } from './sep/SepTheme';

const THEMES = ['base', 'pmm', 'sep'] as const;
const MODES: PaletteMode[] = ['light', 'dark'];
const cases = THEMES.flatMap((name) =>
  MODES.map((mode) => [name, mode] as const)
);
const themes = new Map(
  cases.map(([name, mode]) => [
    `${name}-${mode}`,
    createTheme(getThemeOptions(name)(mode)),
  ])
);
const themeFor = (name: string, mode: PaletteMode) =>
  themes.get(`${name}-${mode}`)!;
const chartTokens = {
  base: { light: semanticTokensLight.charts, dark: semanticTokensDark.charts },
  pmm: { light: semanticTokensLight.charts, dark: semanticTokensDark.charts },
  sep: { light: sepTokensLight.charts, dark: sepTokensDark.charts },
};

// Light-mode chart tokens below 3:1 (WCAG 1.4.11); flagged to design, not to be "fixed" here.
const KNOWN_LIGHT_FAILURES = [
  'chart1',
  'chart2',
  'chart3',
  'chart4',
  'chart5',
  'chart7',
];

describe('palette.charts', () => {
  it.each(cases)(
    'carries the seven chart color tokens in %s %s',
    (name, mode) => {
      const { charts } = themeFor(name, mode).palette;

      const tokens = chartTokens[name][mode];

      expect(Object.keys(charts)).toEqual([...CHART_SERIES_COLORS]);
      CHART_SERIES_COLORS.forEach((key) =>
        expect(charts[key], key).toBe(tokens[key])
      );
    }
  );
});

describe('chart color contrast', () => {
  it.each(cases)(
    'primary marks hold 3:1 on both surfaces in %s %s',
    (name, mode) => {
      const { primary, background } = themeFor(name, mode).palette;

      expect(
        getContrastRatio(primary.main, background.default)
      ).toBeGreaterThanOrEqual(3);
      expect(
        getContrastRatio(primary.main, background.paper)
      ).toBeGreaterThanOrEqual(3);
    }
  );

  it.each(THEMES)(
    'every chart color holds 3:1 on both surfaces in %s dark',
    (name) => {
      const { charts, background } = themeFor(name, 'dark').palette;

      CHART_SERIES_COLORS.forEach((key) => {
        expect(
          getContrastRatio(charts[key], background.default),
          key
        ).toBeGreaterThanOrEqual(3);
        expect(
          getContrastRatio(charts[key], background.paper),
          key
        ).toBeGreaterThanOrEqual(3);
      });
    }
  );

  it.each(THEMES)(
    'only the known light chart colors fall below 3:1 in %s light',
    (name) => {
      const { charts, background } = themeFor(name, 'light').palette;
      const failing = CHART_SERIES_COLORS.filter(
        (key) =>
          getContrastRatio(charts[key], background.default) < 3 ||
          getContrastRatio(charts[key], background.paper) < 3
      );

      expect(failing).toEqual(KNOWN_LIGHT_FAILURES);
    }
  );

  it.each(cases)(
    'text on any chart color holds 4.5:1 in %s %s',
    (name, mode) => {
      const theme = themeFor(name, mode);
      const fills = [
        theme.palette.primary.main,
        ...CHART_SERIES_COLORS.map((key) => theme.palette.charts[key]),
      ];

      fills.forEach((fill) =>
        expect(
          getContrastRatio(getReadableTextColor(theme, fill), fill),
          fill
        ).toBeGreaterThanOrEqual(4.5)
      );
    }
  );
});

describe('@mui/x-charts theme defaults', () => {
  const style = (override: unknown, theme: Theme) =>
    (override as (props: { theme: Theme }) => CSSObject)({ theme });

  it.each(cases)(
    'series default to the chart colors in order in %s %s',
    (name, mode) => {
      const theme = themeFor(name, mode);

      expect(
        theme.components?.MuiChartsDataProvider?.defaultProps?.colors
      ).toEqual(CHART_SERIES_COLORS.map((key) => theme.palette.charts[key]));
    }
  );

  it.each(cases)(
    'tooltips share the inverted tooltip surface in %s %s',
    (name, mode) => {
      const theme = themeFor(name, mode);
      const chartTooltip = theme.components?.MuiChartsTooltip?.styleOverrides;
      const paper = style(chartTooltip?.paper, theme);
      const cell = style(chartTooltip?.cell, theme);
      const tooltip = style(
        theme.components?.MuiTooltip?.styleOverrides?.tooltip,
        theme
      );

      expect(paper.backgroundColor).toBe(tooltip.backgroundColor);
      expect(paper.color).toBe(tooltip.color);
      expect(
        getContrastRatio(cell.color as string, paper.backgroundColor as string)
      ).toBeGreaterThanOrEqual(4.5);
    }
  );
});
