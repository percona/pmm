import { createContext } from 'react';

type ColorModeContextProps = {
  colorMode: 'light' | 'dark';
  toggleColorMode: () => void;
};
export const ColorModeContext = createContext<ColorModeContextProps>({
  colorMode: 'light',
  toggleColorMode: () => {},
});
