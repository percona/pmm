import { useState, useMemo, useCallback } from 'react';
import { ThemeProvider } from '@mui/material';
import { PaletteMode, createTheme } from '@mui/material';
import CssBaseline from '@mui/material/CssBaseline';
import { ThemeContextProviderProps } from './theme-context-provider.types';
import { ColorModeContext } from './theme-contexts';

const COLOR_MODE_STORAGE_KEY = 'colorMode';

const getColorModeFromLocalStorage = (): PaletteMode => {
  const colorMode = localStorage.getItem(COLOR_MODE_STORAGE_KEY);

  if (colorMode && (colorMode === 'light' || colorMode === 'dark')) {
    return colorMode;
  }

  return 'light';
};

const ThemeContextProvider = ({
  children,
  themeOptions,
  saveColorModeOnLocalStorage,
}: ThemeContextProviderProps) => {

  const [colorMode, setColorMode] = useState<PaletteMode>(
    saveColorModeOnLocalStorage ? getColorModeFromLocalStorage() : 'light'
  );
  const toggleColorMode = useCallback(() => {
    setColorMode((prevMode) => {
      const newColorMode = prevMode === 'light' ? 'dark' : 'light';
      if (saveColorModeOnLocalStorage) {
        localStorage.setItem(COLOR_MODE_STORAGE_KEY, newColorMode);
      }
      return newColorMode;
    });
  }, [saveColorModeOnLocalStorage]);

  const theme = useMemo(
    () => createTheme(themeOptions(colorMode)),
    [colorMode, themeOptions]
  );

  return (
    <ColorModeContext.Provider value={{ colorMode, toggleColorMode }}>
      <ThemeProvider theme={theme}>
        <CssBaseline />
        {children}
      </ThemeProvider>
    </ColorModeContext.Provider>
  );
};

export default ThemeContextProvider;
