import React from 'react';
import { type Meta, type StoryObj } from '@storybook/react';
import * as DocBlock from '@storybook/addon-docs/blocks';
import Box from '@mui/material/Box';
import CodeBlock from './code-block';
import type { CodeBlockProps } from './code-block.types';

const SAMPLE = `helm install percona/percona-db \\
  --create-namespace \\
  --namespace percona`;

const meta: Meta<CodeBlockProps> = {
  title: 'Data display/Code Block',
  component: CodeBlock as React.ComponentType<CodeBlockProps>,
  tags: ['autodocs'],
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component:
          'Multi-line code block rendered as semantic `<pre><code>`. Theme-aware across light/dark and the base/pmm/sep themes. Pass `copyable` to show a copy button that reuses `CopyToClipboardButton`. Extends MUI `BoxProps`.',
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
      control: 'text',
      description: 'The code content to render. Use a string so it can be copied.',
    },
    copyable: {
      control: 'boolean',
      description: 'Shows a copy-to-clipboard button in the top-right corner.',
      table: { defaultValue: { summary: 'false' } },
    },
    showCopyButtonText: {
      control: 'boolean',
      description: 'When copyable, renders a labeled button instead of an icon-only button.',
      table: { defaultValue: { summary: 'false' } },
    },
    value: {
      control: 'text',
      description:
        'Overrides the text copied to the clipboard. Useful when children are not a plain string.',
    },
    sx: {
      control: false,
      description: 'MUI System prop for style overrides.',
    },
  },
  args: {
    children: SAMPLE,
    copyable: false,
    showCopyButtonText: false,
  },
  decorators: [
    (Story) => (
      <Box sx={{ width: 480, maxWidth: '100%' }}>
        <Story />
      </Box>
    ),
  ],
};

export default meta;
type Story = StoryObj<CodeBlockProps>;

export const Playground: Story = {};

export const Plain: Story = {
  parameters: {
    docs: {
      description: { story: 'A block of code with no copy affordance.' },
    },
  },
  args: { copyable: false },
};

export const Copyable: Story = {
  parameters: {
    docs: {
      description: {
        story: 'Set `copyable` to expose an icon button that copies the block content.',
      },
    },
  },
  args: { copyable: true },
};

export const CopyableWithLabel: Story = {
  name: 'Copyable with label',
  parameters: {
    docs: {
      description: {
        story: 'Combine `copyable` with `showCopyButtonText` for a labeled copy button.',
      },
    },
  },
  args: { copyable: true, showCopyButtonText: true },
};

export const Overflowing: Story = {
  parameters: {
    docs: {
      description: {
        story: 'Lines longer than the container scroll horizontally rather than wrapping.',
      },
    },
  },
  args: {
    copyable: true,
    children:
      "kubectl get pods --all-namespaces -o jsonpath=\"{range .items[*]}{.metadata.name}{'\\t'}{.status.phase}{'\\n'}{end}\"",
  },
};
