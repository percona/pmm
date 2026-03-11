import { render } from '@testing-library/react';
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { createTheme, ThemeProvider } from '@mui/material/styles';
import { ThemeClass } from './ThemeClass';
import { LIGHT_THEME_CLASS, DARK_THEME_CLASS } from './ThemeClass.constants';

describe('ThemeClass', () => {
  beforeEach(() => {
    // Clear any existing classes on body before each test
    document.body.classList.remove(LIGHT_THEME_CLASS, DARK_THEME_CLASS);
  });

  afterEach(() => {
    // Clean up after each test
    document.body.classList.remove(LIGHT_THEME_CLASS, DARK_THEME_CLASS);
  });

  it('applies light theme class when mode is light', () => {
    const lightTheme = createTheme({ palette: { mode: 'light' } });

    render(
      <ThemeProvider theme={lightTheme}>
        <ThemeClass />
      </ThemeProvider>
    );

    expect(document.body.classList.contains(LIGHT_THEME_CLASS)).toBe(true);
    expect(document.body.classList.contains(DARK_THEME_CLASS)).toBe(false);
  });

  it('applies dark theme class when mode is dark', () => {
    const darkTheme = createTheme({ palette: { mode: 'dark' } });

    render(
      <ThemeProvider theme={darkTheme}>
        <ThemeClass />
      </ThemeProvider>
    );

    expect(document.body.classList.contains(DARK_THEME_CLASS)).toBe(true);
    expect(document.body.classList.contains(LIGHT_THEME_CLASS)).toBe(false);
  });

  it('removes old class when theme changes from light to dark', () => {
    const lightTheme = createTheme({ palette: { mode: 'light' } });
    const darkTheme = createTheme({ palette: { mode: 'dark' } });

    const { rerender } = render(
      <ThemeProvider theme={lightTheme}>
        <ThemeClass />
      </ThemeProvider>
    );

    expect(document.body.classList.contains(LIGHT_THEME_CLASS)).toBe(true);

    rerender(
      <ThemeProvider theme={darkTheme}>
        <ThemeClass />
      </ThemeProvider>
    );

    expect(document.body.classList.contains(DARK_THEME_CLASS)).toBe(true);
    expect(document.body.classList.contains(LIGHT_THEME_CLASS)).toBe(false);
  });

  it('removes old class when theme changes from dark to light', () => {
    const lightTheme = createTheme({ palette: { mode: 'light' } });
    const darkTheme = createTheme({ palette: { mode: 'dark' } });

    const { rerender } = render(
      <ThemeProvider theme={darkTheme}>
        <ThemeClass />
      </ThemeProvider>
    );

    expect(document.body.classList.contains(DARK_THEME_CLASS)).toBe(true);

    rerender(
      <ThemeProvider theme={lightTheme}>
        <ThemeClass />
      </ThemeProvider>
    );

    expect(document.body.classList.contains(LIGHT_THEME_CLASS)).toBe(true);
    expect(document.body.classList.contains(DARK_THEME_CLASS)).toBe(false);
  });

  it('removes classes on unmount', () => {
    const lightTheme = createTheme({ palette: { mode: 'light' } });

    const { unmount } = render(
      <ThemeProvider theme={lightTheme}>
        <ThemeClass />
      </ThemeProvider>
    );

    expect(document.body.classList.contains(LIGHT_THEME_CLASS)).toBe(true);

    unmount();

    expect(document.body.classList.contains(LIGHT_THEME_CLASS)).toBe(false);
    expect(document.body.classList.contains(DARK_THEME_CLASS)).toBe(false);
  });
});
