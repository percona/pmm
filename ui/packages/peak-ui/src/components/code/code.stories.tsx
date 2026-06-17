import React from 'react';
import { type Meta, type StoryObj } from '@storybook/react';
import * as DocBlock from '@storybook/addon-docs/blocks';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import { createTheme, ThemeProvider } from '@mui/material/styles';
import Code from './code';
import type { CodeProps } from './code.types';
import { getThemeOptions } from '../../design';

const THEMES = ['base', 'pmm', 'sep'] as const;
const MODES = ['light', 'dark'] as const;

const meta: Meta<CodeProps> = {
  title: 'Data display/Code',
  component: Code as React.ComponentType<CodeProps>,
  tags: ['autodocs'],
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component:
          'Inline code snippet rendered as a semantic `<code>` element. Theme-aware across light/dark and the base/pmm/sep themes. Extends MUI `BoxProps` — `sx`, `className`, and standard HTML attributes are available.',
      },
      page: () => (
        <>
          <DocBlock.Title />
          <DocBlock.Subtitle />
          <DocBlock.Description />
          <DocBlock.Primary />
          <DocBlock.Controls />
          <DocBlock.Stories />
        </>
      ),
    },
  },
  argTypes: {
    children: {
      name: 'content',
      control: 'text',
      description: 'The code content to render inline.',
    },
    sx: {
      control: false,
      description: 'MUI System prop for style overrides.',
    },
  },
  args: {
    children: 'pnpm install',
  },
};

export default meta;
type Story = StoryObj<CodeProps>;

export const Playground: Story = {};

export const InSentence: Story = {
  name: 'In a sentence',
  parameters: {
    docs: {
      description: {
        story: 'Inline code flows naturally within surrounding text.',
      },
    },
  },
  render: () => (
    <Typography variant="body1">
      Run <Code>pnpm build</Code> to compile the library, then import <Code>{'{ Code }'}</Code> from{' '}
      <Code>@percona/percona-ui</Code>.
    </Typography>
  ),
};

export const Wrapping: Story = {
  parameters: {
    docs: {
      description: {
        story: 'Long inline code wraps and breaks within a constrained container.',
      },
    },
  },
  render: () => (
    <Box sx={{ maxWidth: 280 }}>
      <Typography variant="body2">
        Connect with <Code>mongodb://admin:password@localhost:27017/?authSource=admin</Code>.
      </Typography>
    </Box>
  ),
};

const ThemeCell = ({
  themeName,
  mode,
}: {
  themeName: (typeof THEMES)[number];
  mode: (typeof MODES)[number];
}) => {
  const theme = createTheme(getThemeOptions(themeName)(mode));
  return (
    <ThemeProvider theme={theme}>
      <Box
        sx={{
          p: 2,
          borderRadius: 1,
          backgroundColor: theme.palette.background.default,
          border: `1px solid ${theme.palette.divider}`,
        }}
      >
        <Typography variant="body2">
          Run <Code>pnpm build</Code>
        </Typography>
      </Box>
    </ThemeProvider>
  );
};

export const Themes: Story = {
  parameters: {
    layout: 'padded',
    docs: {
      description: {
        story:
          'Inline code uses the active theme’s brand color, so the look changes with the base / pmm / sep theme and the light / dark mode. Use the toolbar to switch, or compare them side by side here.',
      },
    },
  },
  render: () => (
    <Box
      sx={{ display: 'grid', gridTemplateColumns: 'auto 1fr 1fr', gap: 2, alignItems: 'center' }}
    >
      <Box />
      {MODES.map((mode) => (
        <Typography key={mode} variant="overline">
          {mode}
        </Typography>
      ))}
      {THEMES.map((themeName) => (
        <React.Fragment key={themeName}>
          <Typography variant="overline">{themeName}</Typography>
          {MODES.map((mode) => (
            <ThemeCell key={mode} themeName={themeName} mode={mode} />
          ))}
        </React.Fragment>
      ))}
    </Box>
  ),
};
