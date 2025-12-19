import { type Meta, type StoryObj } from '@storybook/react';
import { Box, IconButton, IconButtonProps } from '@mui/material';
import * as DocBlock from '@storybook/blocks';
import { CachedOutlined } from '@mui/icons-material';

const meta = {
  title: 'Button (Icon Button)',
  component: IconButton,
  tags: ['autodocs'],
  parameters: {
    layout: 'centered',
    docs: {
      source: {
        code: null,
      },
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
    color: {
      options: ['inherit', 'default', 'primary', 'secondary'],
      control: { type: 'inline-radio' },
    },
  },
  render: function Render({ color }) {
    return (
      <Box display="flex" alignItems="center" gap="10px">
        <IconButton color={color} size="large">
          <CachedOutlined />
        </IconButton>
        <IconButton color={color} size="medium">
          <CachedOutlined />
        </IconButton>
        <IconButton color={color} size="small">
          <CachedOutlined />
        </IconButton>
      </Box>
    );
  },
} satisfies Meta<IconButtonProps>;

export default meta;
type Story = StoryObj<IconButtonProps>;

export const Primary: Story = {
  args: {
    color: 'primary',
  },
};

export const Secondary: Story = {
  args: {
    color: 'secondary',
  },
};
