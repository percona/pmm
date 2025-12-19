import { type Meta, type StoryObj } from '@storybook/react';
import { Box, Button, ButtonProps } from '@mui/material';
import * as DocBlock from '@storybook/blocks';
import { ArrowBack, ArrowForward } from '@mui/icons-material';

const meta = {
  title: 'Button',
  component: Button,
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
          <DocBlock.Primary />
          <DocBlock.Controls />
          <DocBlock.Stories />
        </>
      ),
    },
  },
  args: {
    size: 'large',
    variant: 'contained',
  },
  argTypes: {
    variant: {
      options: ['contained', 'text', 'outlined'],
      control: { type: 'inline-radio' },
    },
    size: {
      options: ['large', 'medium', 'small'],
      control: { type: 'inline-radio' },
    },
  },
  render: function Render({ variant, size, ...args }) {
    return (
      <Box display="flex" alignItems="center" gap="10px">
        <Button variant={variant} size={size} {...args}>
          Large
        </Button>
      </Box>
    );
  },
} satisfies Meta<ButtonProps>;

export default meta;
type Story = StoryObj<ButtonProps>;

export const Contained: Story = {
  args: {
    variant: 'contained',
  },
};

export const Outlined: Story = {
  args: {
    variant: 'outlined',
  },
};

export const Text: Story = {
  args: {
    variant: 'text',
  },
};

export const Hovered: Story = {
  parameters: {
    pseudo: {
      hover: true,
    },
  },

  render: function Render({ size }) {
    return (
      <Box display="flex" alignItems="center" gap="10px">
        <Button variant="contained" size={size}>
          Large
        </Button>
        <Button variant="outlined" size={size}>
          Large
        </Button>
        <Button variant="text" size={size}>
          Large
        </Button>
      </Box>
    );
  },
};

export const Disabled: Story = {
  render: function Render({ size }) {
    return (
      <Box display="flex" alignItems="center" gap="10px">
        <Button variant="contained" size={size} disabled={true}>
          Large
        </Button>
        <Button variant="outlined" size={size} disabled={true}>
          Large
        </Button>
        <Button variant="text" size={size} disabled={true}>
          Large
        </Button>
      </Box>
    );
  },
};

export const WithIcon: Story = {
  args: {
    variant: 'contained',
    startIcon: <ArrowBack />,
    endIcon: <ArrowForward />,
  },
  render: function Render({ variant, size, startIcon, endIcon }) {
    return (
      <Box display="flex" alignItems="center" gap="10px">
        <Button startIcon={startIcon} variant={variant} size={size}>
          Large
        </Button>
        <Button endIcon={endIcon} variant={variant} size={size}>
          Large
        </Button>
      </Box>
    );
  },
};
