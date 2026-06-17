import React from 'react';
import { type Meta, type StoryObj } from '@storybook/react';
import * as DocBlock from '@storybook/addon-docs/blocks';
import Box from '@mui/material/Box';
import CodeBlock from './code-block';
import type { CodeBlockProps } from './code-block.types';

const CodeBlockComp = CodeBlock as React.ComponentType<CodeBlockProps>;

const SAMPLE = `helm install percona/percona-db \\
  --create-namespace \\
  --namespace percona`;

const meta: Meta<CodeBlockProps> = {
  title: 'Data display/Code Block',
  component: CodeBlockComp,
  tags: ['autodocs'],
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component: [
          'Multi-line code block rendered as semantic `<pre><code>`. Theme-aware across light/dark and the base/pmm/sep themes. Pass `copyable` to show a copy button that reuses `CopyToClipboardButton`. Extends MUI `BoxProps`.',
          '',
          'Set `language` to enable syntax highlighting (SQL, JS, YAML, …). Highlighting is powered by `prism-react-renderer`, loaded lazily and only when `language` is set — it is an **optional peer dependency**, so without it the block falls back to plain text. Pick a `colorScheme` (defaults to GitHub in light mode, VS Dark in dark mode).',
        ].join('\n'),
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
    language: {
      control: 'text',
      description:
        'Prism language id (e.g. `sql`, `javascript`, `yaml`). When set, the block is syntax-highlighted.',
    },
    colorScheme: {
      control: 'select',
      options: [
        'github',
        'vsLight',
        'vsDark',
        'oneLight',
        'oneDark',
        'dracula',
        'nightOwl',
        'nightOwlLight',
        'oceanicNext',
        'palenight',
        'okaidia',
      ],
      description:
        'Highlighting color scheme. Defaults to `github` (light mode) / `vsDark` (dark mode). Accepts a custom `PrismTheme` object too.',
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

const SQL_SAMPLE = `SELECT id, name, created_at
FROM users
WHERE status = 'active'
ORDER BY created_at DESC
LIMIT 10;`;

export const Highlighted: Story = {
  parameters: {
    docs: {
      description: {
        story:
          'Set `language` to syntax-highlight. The default scheme follows the color mode (GitHub in light, VS Dark in dark).',
      },
    },
  },
  args: {
    language: 'sql',
    copyable: true,
    children: SQL_SAMPLE,
  },
};

export const ColorSchemes: Story = {
  name: 'Color schemes',
  parameters: {
    docs: {
      description: {
        story:
          'Pick any bundled scheme via `colorScheme`, including popular ones like GitHub and Dracula.',
      },
    },
  },
  render: (args) => (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
      {(['github', 'dracula', 'nightOwl', 'oneDark'] as const).map((scheme) => (
        <CodeBlockComp key={scheme} {...args} language="sql" colorScheme={scheme} copyable>
          {`-- ${scheme}\n${SQL_SAMPLE}`}
        </CodeBlockComp>
      ))}
    </Box>
  ),
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
