import React from 'react';
import { type Meta, type StoryObj } from '@storybook/react';
import * as DocBlock from '@storybook/addon-docs/blocks';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import Code from './code';
import type { CodeProps } from './code.types';

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
    content: {
      control: 'text',
      description: 'The code content to render inline.',
    },
    sx: {
      control: false,
      description: 'MUI System prop for style overrides.',
    },
  },
  args: {
    content: 'pnpm install',
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
      Run <Code content="pnpm build" /> to compile the library, then import{' '}
      <Code content="{ Code }" /> from <Code content="@percona/percona-ui" />.
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
        Connect with <Code content="mongodb://admin:password@localhost:27017/?authSource=admin" />.
      </Typography>
    </Box>
  ),
};
