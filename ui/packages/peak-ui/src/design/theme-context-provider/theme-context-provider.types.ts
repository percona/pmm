import { PaletteMode, ThemeOptions } from '@mui/material';

export type ThemeContextProviderProps = {
  /** The app tree that receives the theme. */
  children: React.ReactNode;
  /** Theme to apply, as a function of the color mode, for example `baseThemeOptions`, `pmmThemeOptions`, or `sepThemeOptions`. */
  themeOptions: (mode: PaletteMode) => ThemeOptions;
  /** Remembers the user's light/dark choice in the browser so it survives reloads. */
  saveColorModeOnLocalStorage?: boolean;
};
