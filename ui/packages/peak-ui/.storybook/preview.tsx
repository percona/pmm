import React, { useMemo } from "react";
import type { Preview } from "@storybook/react";
import { CssBaseline, createTheme, ThemeProvider } from "@mui/material";
import { getThemeOptions } from "../src/design";

const preview: Preview = {
  decorators: [
    (Story, context) => {
      const {
        globals: { mode, theme: themeName },
      } = context;

      const theme = useMemo(
        () => createTheme(getThemeOptions(themeName)(mode)),
        [mode, themeName]
      );

      document.body.style.backgroundColor = theme.palette.background.default;

      return (
        <ThemeProvider theme={theme}>
          <CssBaseline />
          <Story />
        </ThemeProvider>
      );
    },
  ],
  globalTypes: {
    mode: {
      name: "mode",
      description: "Color mode",
      defaultValue: "light",
      toolbar: {
        icon: "circlehollow",
        items: [
          { value: "light", icon: "circlehollow", title: "light" },
          { value: "dark", icon: "circle", title: "dark" },
        ],
        showName: true,
        dynamicTitle: true,
      },
    },
    theme: {
      name: "theme",
      description: "Theme",
      defaultValue: "base",
      toolbar: {
        icon: "palette",
        items: [
          { value: "base", icon: "palette", title: "Base Theme" },
          { value: "everest", icon: "palette", title: "Everest Theme" },
          { value: "pmm", icon: "palette", title: "PMM Theme" },
        ],
        showName: true,
        dynamicTitle: true,
      },
    },
  },
  parameters: {
    actions: { argTypesRegex: "^on[A-Z].*" },
    controls: {
      matchers: {
        color: /(background|color)$/i,
        date: /Date$/i,
      },
    },
    options: {
      storySort: {
        order: ['Foundations', '*'],
      },
    },
  },
};

export default preview;
