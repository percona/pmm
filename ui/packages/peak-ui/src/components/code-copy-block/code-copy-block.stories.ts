import { type Meta, type StoryObj } from '@storybook/react';
import CodeCopyBlock from './code-copy-block';
import { CodeCopyBlockProps } from './code-copy-block.types';

const meta = {
  title: 'Inputs/Buttons/CodeCopyBlock',
  component: CodeCopyBlock,
  parameters: {
    layout: 'centered',
  },
} satisfies Meta<CodeCopyBlockProps>;

export default meta;
type Story = StoryObj<CodeCopyBlockProps>;

export const WithCopyButtonCommand: Story = {
  args: {
    message:
      'helm install everest percona/everest-db-namespace --create-namespace --namespace <NAMESPACE>',
    showCopyButtonText: true,
  },
};
export const WithoutCopyButtonCommand: Story = {
  args: {
    message:
      'helm install everest percona/everest-db-namespace --create-namespace --namespace <NAMESPACE>',
    showCopyButtonText: false,
  },
};
