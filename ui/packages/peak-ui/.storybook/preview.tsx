import { useMemo } from "react";
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
  initialGlobals: {
    mode: "light",
    theme: "base",
  },
  globalTypes: {
    mode: {
      name: "mode",
      description: "Color mode",
      toolbar: {
        icon: "sun",
        items: [
          { value: "light", icon: "sun", title: "Light mode" },
          { value: "dark", icon: "moon", title: "Dark mode" },
        ],
        dynamicTitle: true,
      },
    },
    theme: {
      name: "theme",
      description: "GUI theme",
      toolbar: {
        icon: "circlehollow",
        items: [
          { value: "base", icon: "circlehollow", title: "Base theme" },
          { value: "pmm", icon: "hearthollow", title: "PMM theme" },
          { value: "sep", icon: "database", title: "SEP theme" },
        ],
        dynamicTitle: true,
      },
    },
  },
  parameters: {
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
          'Inputs', ['Icon Button', 'Text Field', 'Copy to Clipboard'],
          'Data display', ['Chip', 'Code Block', 'Table', 'Tooltip'],
          'Feedback', ['Status Icon'],
          'Navigation',
          'To be reviewed',
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
