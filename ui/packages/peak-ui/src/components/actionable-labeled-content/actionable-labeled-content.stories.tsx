import { type Meta, type StoryObj } from '@storybook/react';
import * as DocBlock from '@storybook/blocks';
import ActionableLabeledContent, {
  ActionableLabeledContentProps,
} from './actionable-labeled-content';
import { TextField } from '@mui/material';

const meta = {
  title: 'ActionableLabeledContent',
  component: ActionableLabeledContent,
  tags: ['autodocs'],
  parameters: {
    layout: 'padded',
    docs: {
      page: () => (
        <>
          <DocBlock.Title />
          <DocBlock.Subtitle />
          <DocBlock.Description />
          <DocBlock.Stories />
        </>
      ),
    },
  },
  argTypes: {
    techPreview: {
      options: [true, false],
      control: { type: 'boolean' },
    },
    caption: {
      control: { type: 'text' },
    },
  },
  render: function Render({ ...props }) {
    return (
      <ActionableLabeledContent {...props} label="My Label">
        <TextField id="outlined-basic" label="Lorem Ipsum" />
      </ActionableLabeledContent>
    );
  },
} satisfies Meta<ActionableLabeledContentProps>;

export default meta;
type Story = StoryObj<ActionableLabeledContentProps>;

export const WithTechPreview: Story = {
  args: {
    techPreview: true,
  },
};

export const WithCaption: Story = {
  args: {
    caption: 'This is a caption',
  },
};

export const WithActionButton: Story = {
  args: {
    actionButtonProps: {
      buttonText: 'My Button',
    },
  },
};

export const WithActionButtonAndTechPreview: Story = {
  args: {
    techPreview: true,
    caption: 'This is a caption',
    actionButtonProps: {
      buttonText: 'My Button',
    },
  },
};
