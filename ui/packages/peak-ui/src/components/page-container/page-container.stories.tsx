import { type Meta, type StoryObj } from '@storybook/react';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import PageContainer from './page-container';

const Filler = ({ label }: { label: string }) => (
  <Box
    sx={{
      p: 3,
      border: '1px dashed',
      borderColor: 'divider',
      borderRadius: 1,
    }}
  >
    <Typography variant="body1">{label}</Typography>
  </Box>
);

const meta: Meta<typeof PageContainer> = {
  title: 'Layout/PageContainer',
  component: PageContainer,
  tags: ['autodocs', 'needs-review'],
  parameters: {
    layout: 'fullscreen',
  },
  argTypes: {
    maxWidth: {
      control: { type: 'select' },
      options: [1000, 600, 1400, 'full'],
      description: "Pixel number (px), or 'full' for 100% width.",
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    children: <Filler label="Default max width (1000px), centered." />,
  },
};

export const Narrow: Story = {
  args: {
    maxWidth: 600,
    children: <Filler label="maxWidth = 600" />,
  },
};

export const Wide: Story = {
  args: {
    maxWidth: 1400,
    children: <Filler label="maxWidth = 1400" />,
  },
};

export const FullWidth: Story = {
  args: {
    maxWidth: 'full',
    children: <Filler label="maxWidth = 'full' (edge to edge with side padding)" />,
  },
};
