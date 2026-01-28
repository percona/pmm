import { useTheme } from '@mui/material';
import { useEffect } from 'react';
import { DARK_THEME_CLASS, LIGHT_THEME_CLASS } from './ThemeClass.constants';

export const ThemeClass = () => {
  const theme = useTheme();
  const mode = theme.palette.mode;

  useEffect(() => {
    const body = document.body;

    body.classList.remove(LIGHT_THEME_CLASS, DARK_THEME_CLASS);

    if (mode === 'dark') {
      body.classList.add(DARK_THEME_CLASS);
    } else {
      body.classList.add(LIGHT_THEME_CLASS);
    }

    return () => {
      body.classList.remove(LIGHT_THEME_CLASS, DARK_THEME_CLASS);
    };
  }, [mode]);

  return null;
};
