import { createContext } from 'react';

type ColorModeContextProps = {
  /** Mode the app is currently in. */
  colorMode: 'light' | 'dark';
  /** Switches between light and dark mode, for a theme toggle control. */
  toggleColorMode: () => void;
};
export const ColorModeContext = createContext<ColorModeContextProps>({
  colorMode: 'light',
  toggleColorMode: () => {},
});
