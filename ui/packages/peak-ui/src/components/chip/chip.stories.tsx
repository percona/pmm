import { type Meta, type StoryObj } from '@storybook/react';
import { Avatar, Box, Typography } from '@mui/material';
import * as DocBlock from '@storybook/blocks';
import Chip from './chip';
import type { ChipProps } from './chip.types';

type ChipColor = NonNullable<ChipProps['color']>;
type ChipVariant = NonNullable<ChipProps['variant']>;

const COLORS: ChipColor[] = ['default', 'success', 'error', 'info', 'warning'];

const VARIANTS: ChipVariant[] = ['filled', 'outlined'];

type PlaygroundArgs = ChipProps & {
  withAvatar?: boolean;
  withDelete?: boolean;
  clickable?: boolean;
};

const rowSx = {
  display: 'flex',
  alignItems: 'center',
  gap: '8px',
  flexWrap: 'wrap' as const,
};

const stackSx = {
  display: 'flex',
  flexDirection: 'column' as const,
  gap: '16px',
};

const meta = {
  title: 'Data display/Chip',
  component: Chip,
  tags: ['autodocs'],
  parameters: {
    layout: 'centered',
    docs: {
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
    label: 'Chip',
    variant: 'filled',
    color: 'default',
    size: 'medium',
  },
  argTypes: {
    variant: {
      options: VARIANTS,
      control: { type: 'inline-radio' },
    },
    color: {
      options: COLORS,
      control: { type: 'inline-radio' },
    },
    size: {
      options: ['small', 'medium'],
      control: { type: 'inline-radio' },
    },
  },
} satisfies Meta<PlaygroundArgs>;

export default meta;
type Story = StoryObj<PlaygroundArgs>;

export const Playground: Story = {
  args: {
    withAvatar: false,
    withDelete: false,
    clickable: false,
  },
  argTypes: {
    withAvatar: { control: { type: 'boolean' }, name: 'avatar' },
    withDelete: { control: { type: 'boolean' }, name: 'onDelete' },
    clickable: { control: { type: 'boolean' }, name: 'clickable' },
  },
  render: function Render({ withAvatar, withDelete, clickable, ...rest }) {
    return (
      <Chip
        {...rest}
        avatar={withAvatar ? <Avatar>P</Avatar> : undefined}
        onDelete={withDelete ? () => undefined : undefined}
        onClick={clickable ? () => undefined : undefined}
      />
    );
  },
};

export const Variants: Story = {
  render: function Render(args) {
    return (
      <Box sx={rowSx}>
        {VARIANTS.map((variant) => (
          <Chip key={variant} {...args} variant={variant} label={variant} />
        ))}
      </Box>
    );
  },
};

export const Colors: Story = {
  render: function Render(args) {
    return (
      <Box sx={stackSx}>
        {VARIANTS.map((variant) => (
          <Box key={variant} sx={stackSx}>
            <Typography variant="overline">{variant}</Typography>
            <Box sx={rowSx}>
              {COLORS.map((color) => (
                <Chip key={color} {...args} variant={variant} color={color} label={color} />
              ))}
            </Box>
          </Box>
        ))}
      </Box>
    );
  },
};

export const Sizes: Story = {
  render: function Render(args) {
    return (
      <Box sx={stackSx}>
        {VARIANTS.map((variant) => (
          <Box key={variant} sx={rowSx}>
            <Chip {...args} variant={variant} size="small" label={`${variant} / small`} />
            <Chip {...args} variant={variant} size="medium" label={`${variant} / medium`} />
          </Box>
        ))}
      </Box>
    );
  },
};

export const WithAvatar: Story = {
  render: function Render(args) {
    return (
      <Box sx={rowSx}>
        {VARIANTS.map((variant) => (
          <Chip
            key={variant}
            {...args}
            variant={variant}
            label={variant}
            avatar={<Avatar>P</Avatar>}
          />
        ))}
      </Box>
    );
  },
};

export const Deletable: Story = {
  render: function Render(args) {
    return (
      <Box sx={rowSx}>
        {VARIANTS.map((variant) => (
          <Chip
            key={variant}
            {...args}
            variant={variant}
            label={variant}
            onDelete={() => undefined}
          />
        ))}
      </Box>
    );
  },
};

export const Clickable: Story = {
  render: function Render(args) {
    return (
      <Box sx={rowSx}>
        {VARIANTS.map((variant) => (
          <Chip
            key={variant}
            {...args}
            variant={variant}
            label={variant}
            clickable
            onClick={() => undefined}
          />
        ))}
      </Box>
    );
  },
};
