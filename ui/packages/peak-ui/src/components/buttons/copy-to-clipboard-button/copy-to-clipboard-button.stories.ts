import { type Meta, type StoryObj } from '@storybook/react';
import CopyToClipboardButton from './CopyToClipboardButton';
import { CopyToClipboardButtonProps } from './CopyToClipboardButton.types';

const meta = {
  title: 'Inputs/Buttons/CopyToClipboardButton',
  component: CopyToClipboardButton,
  parameters: {
    layout: 'centered',
  },
} satisfies Meta<CopyToClipboardButtonProps>;

export default meta;
type Story = StoryObj<CopyToClipboardButtonProps>;

export const ButtonWithCopyCommand: Story = {
  args: {
    textToCopy: 'text to copy',
    showCopyButtonText: true,
    copyCommand: 'Copy',
  },
};
export const ButtonWithoutCopyButtonCommand: Story = {
  args: {
    textToCopy: 'text to copy',
    showCopyButtonText: false,
  },
};
