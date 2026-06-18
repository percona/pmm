import React from 'react';
import { type Meta, type StoryObj } from '@storybook/react';
import * as DocBlock from '@storybook/addon-docs/blocks';
import Box from '@mui/material/Box';
import CodeBlock from './code-block';
import type { CodeBlockProps, CodeColorScheme } from './code-block.types';

const CodeBlockComp = CodeBlock as React.ComponentType<CodeBlockProps>;

const SAMPLE = `helm install percona/percona-db \\
  --create-namespace \\
  --namespace percona`;

const SQL_SAMPLE = `SELECT id, name, created_at
FROM users
WHERE status = 'active'
ORDER BY created_at DESC
LIMIT 10;`;

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
          'Set `language` to enable syntax highlighting (SQL, JS, YAML, etc.). Highlighting is powered by `prism-react-renderer`. With no `colorScheme` set it follows the color mode (nightOwlLight in light, okaidia in dark). In that logic, chosing a `colorScheme` locks it, regardless of the chosen light/dark mode.',
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
    content: {
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
        'Overrides the text copied to the clipboard. Useful when the content is not a plain string.',
    },
    language: {
      control: 'text',
      description:
        'Prism language id (e.g. `sql`, `javascript`, `yaml`). When set, the block is syntax-highlighted.',
    },
    colorScheme: {
      control: 'select',
      options: [
        'dracula',
        'duotoneDark',
        'duotoneLight',
        'github',
        'gruvboxMaterialDark',
        'gruvboxMaterialLight',
        'jettwaveDark',
        'jettwaveLight',
        'nightOwl',
        'nightOwlLight',
        'oceanicNext',
        'okaidia',
        'oneDark',
        'oneLight',
        'palenight',
        'shadesOfPurple',
        'synthwave84',
        'ultramin',
        'vsDark',
        'vsLight',
      ],
      description:
        'Highlighting color scheme. **Only applies when `language` is set.** Defaults to `nightOwlLight` (light mode) / `okaidia` (dark mode). Accepts a custom `PrismTheme` object too.',
    },
    wrap: {
      control: 'boolean',
      description:
        'Wrap long lines instead of scrolling. Defaults to `false`, where lines longer than the container overflow horizontally with a scrollbar.',
      table: { defaultValue: { summary: 'false' } },
    },
  },
  args: {
    content: SAMPLE,
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

export const Playground: Story = {
  parameters: {
    docs: {
      description: {
        story:
          'Highlighted by default. With no `colorScheme` selected it follows the color mode (nightOwlLight in light, okaidia in dark) — toggle the Storybook light/dark control to see it switch. Pick a `colorScheme` to pin a specific one, or clear `language` for a plain block.',
      },
    },
  },
  args: {
    content: SQL_SAMPLE,
    language: 'sql',
    copyable: true,
  },
};

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

export const Highlighted: Story = {
  parameters: {
    docs: {
      description: {
        story:
          'Set `language` to syntax-highlight. The default scheme follows the color mode (GitHub in light, Okaidia in dark).',
      },
    },
  },
  args: {
    language: 'sql',
    copyable: true,
    content: SQL_SAMPLE,
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
  render: (args) => {
    const allSchemes: CodeColorScheme[] = [
      'github',
      'vsLight',
      'ultramin',
      'duotoneLight',
      'gruvboxMaterialLight',
      'jettwaveLight',
      'nightOwlLight',
      'oneLight',
      'dracula',
      'vsDark',
      'duotoneDark',
      'gruvboxMaterialDark',
      'jettwaveDark',
      'nightOwl',
      'oceanicNext',
      'okaidia',
      'oneDark',
      'palenight',
      'shadesOfPurple',
      'synthwave84',
    ];
    return (
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
        {allSchemes.map((scheme) => (
          <CodeBlockComp
            key={scheme}
            {...args}
            language="sql"
            colorScheme={scheme}
            copyable
            content={`-- ${scheme}\n${SQL_SAMPLE}`}
          />
        ))}
      </Box>
    );
  },
};

export const Overflowing: Story = {
  parameters: {
    docs: {
      description: {
        story:
          'Default behavior: lines longer than the container scroll horizontally rather than wrapping. Set `wrap` to make them wrap instead.',
      },
    },
  },
  args: {
    copyable: true,
    content:
      "kubectl get pods --all-namespaces -o jsonpath=\"{range .items[*]}{.metadata.name}{'\\t'}{.status.phase}{'\\n'}{end}\"",
  },
};

export const Wrapped: Story = {
  parameters: {
    docs: {
      description: {
        story: 'Set `wrap` so long lines wrap to the next line instead of scrolling horizontally.',
      },
    },
  },
  args: {
    copyable: true,
    wrap: true,
    content:
      "kubectl get pods --all-namespaces -o jsonpath=\"{range .items[*]}{.metadata.name}{'\\t'}{.status.phase}{'\\n'}{end}\"",
  },
};
