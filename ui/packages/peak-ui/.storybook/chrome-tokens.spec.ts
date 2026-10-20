import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';
import {
  primitives,
  semanticTokensDark,
  semanticTokensLight,
  shape,
} from '../src/design/themes/base/BaseTheme';
import { CHROME } from './chrome-tokens';

const __dirname = dirname(fileURLToPath(import.meta.url));
const staticDir = join(__dirname, 'static');
const read = (name: string) => readFileSync(join(staticDir, name), 'utf8');
const colorsIn = (svg: string) =>
  new Set((svg.match(/#[0-9A-Fa-f]{6}/g) ?? []).map((c) => c.toUpperCase()));

describe('Storybook chrome tokens', () => {
  it('mirrors the Base theme tokens each value names', () => {
    expect(CHROME.brand).toBe(primitives.primary.purple[500]);
    expect(CHROME.brandDark).toBe(primitives.primary.purple[300]);

    expect(CHROME.textPrimary).toBe(semanticTokensLight.text.primary);
    expect(CHROME.textSecondary).toBe(semanticTokensLight.text.secondary);
    expect(CHROME.textPrimaryDark).toBe(semanticTokensDark.text.primary);

    expect(CHROME.surfaceElevation0).toBe(semanticTokensLight.surfaces.elevation0);
    expect(CHROME.surfaceElevation1).toBe(semanticTokensLight.surfaces.elevation1);
    expect(CHROME.surfaceElevation0Dark).toBe(semanticTokensDark.surfaces.elevation0);
    expect(CHROME.surfaceElevation1Dark).toBe(semanticTokensDark.surfaces.elevation1);

    expect(CHROME.contour).toBe(semanticTokensLight.lines.contour);
    expect(CHROME.divider).toBe(semanticTokensLight.lines.divider);
    expect(CHROME.contourDark).toBe(semanticTokensDark.lines.contour);
    expect(CHROME.dividerDark).toBe(semanticTokensDark.lines.divider);

    expect(CHROME.radiusSm).toBe(shape.borderRadiusSm);
    expect(CHROME.radiusXs).toBe(shape.borderRadiusXs);
  });

  // The manager renders brandImage as an <img>, so CSS cannot reach the logo's fills.
  // Asserting the baked-in hexes keeps them honest instead.
  it('keeps the header logos on brand token colors', () => {
    expect(colorsIn(read('logo-peak-ui.svg'))).toEqual(
      new Set([primitives.primary.purple[500], primitives.primary.black[950]]),
    );
    expect(colorsIn(read('logo-peak-ui-dark.svg'))).toEqual(
      new Set([primitives.primary.purple[300], primitives.primary.black[50]]),
    );
    expect(colorsIn(read('favicon.svg'))).toEqual(new Set([primitives.primary.purple[500]]));
  });

  it('has no pre-rebrand colors left in the chrome stylesheets', () => {
    const legacy = [/#2C323E/i, /#FBFBFB/i, /#029CFD/i, /#2CBEA2/i, /rgba\(38, 85, 115/, /rgba\(44, 50, 62/];
    for (const file of ['manager-head.html', 'preview-head.html']) {
      const css = readFileSync(join(__dirname, file), 'utf8');
      for (const pattern of legacy) {
        expect(css, `${file} still uses ${pattern}`).not.toMatch(pattern);
      }
    }
  });
});
