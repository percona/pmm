import { createContext, useContext } from 'react';

export type ColorMode = 'light' | 'dark';

export type ThemeContextValue = {
  mode: ColorMode;
  setTheme: (mode: ColorMode) => Promise<void>;
};

export const ThemeCtx = createContext<ThemeContextValue | null>(null);

export const usePmmTheme = (): ThemeContextValue => {
  const ctx = useContext(ThemeCtx);
  if (!ctx) throw new Error('usePmmTheme must be used within ThemeProvider');
  return ctx;
};