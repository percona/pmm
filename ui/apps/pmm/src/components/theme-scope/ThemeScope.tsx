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
  children: ReactNode;
}

/** Renders an area of the page, its menus and tooltips included, in another Peak theme, in the page's colour mode. */
export const ThemeScope: FC<ThemeScopeProps> = ({ theme: name, children }) => {
  const { mode } = useTheme().palette;
  const theme = useMemo(
    () => createTheme(SCOPED_THEMES[name](mode)),
    [name, mode]
  );

  return <ThemeProvider theme={theme}>{children}</ThemeProvider>;
};
