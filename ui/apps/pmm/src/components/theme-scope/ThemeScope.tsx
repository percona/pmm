import { createTheme, ThemeProvider, useTheme } from '@mui/material/styles';
import { baseThemeOptions, pmmThemeOptions } from '@percona/peak-ui';
import { type FC, type ReactNode, useMemo } from 'react';

// Base is Peak's neutral theme (black and white primaries); PMM is the app's own (purple).
export const SCOPED_THEMES = {
  base: baseThemeOptions,
  pmm: pmmThemeOptions,
} as const;

export type ScopedThemeName = keyof typeof SCOPED_THEMES;

export interface ThemeScopeProps {
  theme: ScopedThemeName;
  /** `inverse` draws the area in the opposite colour mode to the page's, for something that must stand out from it. */
  mode?: 'page' | 'inverse';
  children: ReactNode;
}

/** Renders an area of the page, its menus and tooltips included, in another Peak theme, in the page's colour mode or its inverse. */
export const ThemeScope: FC<ThemeScopeProps> = ({
  theme: name,
  mode: scopeMode = 'page',
  children,
}) => {
  const pageMode = useTheme().palette.mode;
  const mode =
    scopeMode === 'page' ? pageMode : pageMode === 'dark' ? 'light' : 'dark';
  const theme = useMemo(
    () => createTheme(SCOPED_THEMES[name](mode)),
    [name, mode]
  );

  return <ThemeProvider theme={theme}>{children}</ThemeProvider>;
};
