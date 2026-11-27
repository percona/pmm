// Brand values the Storybook chrome is styled with, mirrored from the Base theme.
// Kept literal (and MUI-free) so the manager bundle stays lean; `chrome-tokens.spec.ts`
// fails if any value drifts from the token it names.
export const CHROME = {
  // primitives.primary.purple[500] / [300]
  brand: '#7056FC',
  brandDark: '#B6B2FF',

  // semanticTokensLight.text.primary / .secondary
  textPrimary: 'rgb(40, 39, 39)',
  textSecondary: 'rgba(40, 39, 39, 0.55)',
  // semanticTokensDark.text.primary
  textPrimaryDark: 'rgb(246, 245, 245)',

  // semanticTokensLight.surfaces.elevation0 / .elevation1
  surfaceElevation0: '#F6F5F5',
  surfaceElevation1: '#FFFFFF',
  // semanticTokensDark.surfaces.elevation0 / .elevation1
  surfaceElevation0Dark: '#3D3C3C',
  surfaceElevation1Dark: '#282727',

  // semanticTokensLight.lines.contour / .divider
  contour: 'rgba(0, 0, 0, 0.08)',
  divider: 'rgba(40, 39, 39, 0.25)',
  // semanticTokensDark.lines.contour / .divider
  contourDark: 'rgba(255, 255, 255, 0.12)',
  dividerDark: 'rgba(255, 255, 255, 0.25)',

  // shape.borderRadiusSm / .borderRadiusXs
  radiusSm: 5,
  radiusXs: 3,
} as const;
