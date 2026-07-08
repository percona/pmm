import { type Meta, type StoryObj } from '@storybook/react';
import * as DocBlock from '@storybook/addon-docs/blocks';
import Stack from '@mui/material/Stack';
import CopyToClipboardButton from './CopyToClipboardButton';
import { CopyToClipboardButtonProps } from './CopyToClipboardButton.types';

const meta: Meta<CopyToClipboardButtonProps> = {
  title: 'Inputs/Copy to Clipboard',
  component: CopyToClipboardButton,
  tags: ['autodocs', 'stable'],
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component:
          'Copies a string to the clipboard and confirms with a tooltip. Renders as an icon button by default, or a labeled button when `showCopyButtonText` is set. In non-secure contexts (no `navigator.clipboard`) the button is disabled and explains why.',
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
    textToCopy: {
      control: 'text',
      description: 'The string written to the clipboard when clicked.',
    },
    showCopyButtonText: {
      control: 'boolean',
      description: 'Renders a labeled button instead of an icon-only button.',
      table: { defaultValue: { summary: 'false' } },
    },
    copyCommand: {
      control: 'text',
      description: 'Label shown when `showCopyButtonText` is enabled.',
      table: { defaultValue: { summary: "'Copy command'" } },
    },
    buttonProps: {
      control: false,
      description: 'Props forwarded to the underlying MUI `Button` / `IconButton`.',
    },
    iconSx: {
      control: false,
      description: 'MUI System prop applied to the copy icon.',
    },
  },
  args: {
    textToCopy: 'helm install percona/percona-db --namespace percona',
  },
};

export default meta;
type Story = StoryObj<CopyToClipboardButtonProps>;

export const Playground: Story = {};

export const IconOnly: Story = {
  name: 'Icon only',
  parameters: {
    docs: {
      description: { story: 'Default appearance: an icon-only button.' },
    },
  },
  args: { showCopyButtonText: false },
};

export const WithLabel: Story = {
  name: 'With label',
  parameters: {
    docs: {
      description: {
        story: 'Set `showCopyButtonText` and customize the text with `copyCommand`.',
      },
    },
  },
  render: (args) => (
    <Stack direction="row" gap={2}>
      <CopyToClipboardButton {...args} showCopyButtonText copyCommand="Copy" />
      <CopyToClipboardButton {...args} showCopyButtonText copyCommand="Copy command" />
    </Stack>
  ),
};
