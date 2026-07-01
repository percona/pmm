import type { Meta, StoryObj } from '@storybook/react';
import { Box, Typography } from '@mui/material';
import { shape } from '../../design/themes/base';

const meta = {
  title: 'Foundations/Shape',
  tags: ['stable'],
  parameters: {
    layout: 'padded',
  },
} satisfies Meta;

export default meta;
type Story = StoryObj;

const tokenNames = Object.keys(shape) as (keyof typeof shape)[];

const labelStyle = {
  display: 'block',
  fontFamily: '"Roboto Mono", monospace',
  fontSize: '0.8125rem',
  marginTop: '0.75rem',
};

const Swatch = ({ name, radius }: { name: string; radius: number }) => (
  <Box sx={{ textAlign: 'center' }}>
    <Box
      sx={{
        width: 96,
        height: 96,
        backgroundColor: 'primary.main',
        borderRadius: `${radius}px`,
      }}
    />
    <Typography component="span" sx={labelStyle}>
      {name}
    </Typography>
    <Typography component="span" variant="helperText" color="text.secondary" display="block">
      {radius}px
    </Typography>
  </Box>
);

export const Scale: Story = {
  tags: ['!dev'],
  render: () => (
    <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 4 }}>
      {tokenNames.map((name) => (
        <Swatch key={name} name={name} radius={shape[name]} />
      ))}
    </Box>
  ),
};

export const FullyRounded: Story = {
  tags: ['!dev'],
  render: () => (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: 4 }}>
      <Box
        sx={{
          width: 96,
          height: 96,
          backgroundColor: 'primary.main',
          borderRadius: `${shape.borderRadiusFull}px`,
        }}
      />
      <Box
        sx={{
          width: 220,
          height: 48,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          backgroundColor: 'primary.main',
          color: 'primary.contrastText',
          borderRadius: `${shape.borderRadiusFull}px`,
        }}
      >
        <Typography variant="menuText">Pill on wider elements</Typography>
      </Box>
    </Box>
  ),
};
