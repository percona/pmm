import React, { useCallback, useMemo, useState } from 'react';
import { ThemeCtx, type ColorMode } from './theme.context';

export const ThemeProvider: React.FC<React.PropsWithChildren> = ({ children }) => {
  const [mode, setMode] = useState<ColorMode>('light');

  const setTheme = useCallback(async (next: ColorMode) => {
    setMode(next); // passive: no side-effects yet
  }, []);

  const value = useMemo(() => ({ mode, setTheme }), [mode, setTheme]);

  return <ThemeCtx.Provider value={value}>{children}</ThemeCtx.Provider>;
};
