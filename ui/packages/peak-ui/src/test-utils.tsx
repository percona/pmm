import type { ReactNode } from 'react';
import { render } from '@testing-library/react';
import { createTheme, ThemeProvider } from '@mui/material/styles';
import { getThemeOptions } from './design';

export const testTheme = createTheme(getThemeOptions('base')('light'));

export const renderInTheme = (node: ReactNode) =>
  render(<ThemeProvider theme={testTheme}>{node}</ThemeProvider>);
