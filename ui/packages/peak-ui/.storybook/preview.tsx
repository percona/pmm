import React, { useMemo } from "react";
import type { Preview } from "@storybook/react";
import { CssBaseline, createTheme, ThemeProvider } from "@mui/material";
import { getThemeOptions } from "../src/design";
import peakDesign from "./peak-design";

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

      const bgColor = theme.palette.background.default;
      document.body.style.backgroundColor = bgColor;

      const content = (
        <ThemeProvider theme={theme}>
          <CssBaseline />
          <Story />
        </ThemeProvider>
      );

      if (context.viewMode === 'docs') {
        return <div style={{ backgroundColor: bgColor, padding: '2rem' }}>{content}</div>;
      }

      return content;
    },
  ],
  globalTypes: {
    mode: {
      name: "mode",
      description: "Color mode",
      defaultValue: "light",
      toolbar: {
        icon: "sun",
        items: [
          { value: "light", icon: "sun", title: "Light mode" },
          { value: "dark", icon: "moon", title: "Dark mode" },
        ],
        showName: true,
        dynamicTitle: true,
      },
    },
    theme: {
      name: "theme",
      description: "GUI theme",
      defaultValue: "base",
      toolbar: {
        icon: "circlehollow",
        items: [
          { value: "base", icon: "circlehollow", title: "Base theme" },
          { value: "pmm", icon: "hearthollow", title: "PMM theme" },
          { value: "sep", icon: "database", title: "SEP theme" },
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
    docs: {
      theme: peakDesign,
      codePanel: true,
    },
    options: {
      storySort: {
        order: [
          'Introduction',
          'Foundations', [
            'Colors', [
              'Primitives',
              'Primary',
              'Structural UI',
              'Messaging',
              'Charts'
            ],
            'Typography',
            'Icons'
          ],
          'Data display', ['Chip'],
          'Navigation',
          '*'
        ],
      },
    },
    backgrounds: {
      disable: true,
      grid: {
        disable: true,
      },
    },
  },
};

export default preview;
