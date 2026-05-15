import { type Meta, type StoryObj } from '@storybook/react';
import Avatar from '@mui/material/Avatar';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import * as DocBlock from '@storybook/blocks';
import Chip from './chip';
import type { ChipProps } from './chip.types';

type ChipColor = NonNullable<ChipProps['color']>;
type ChipVariant = NonNullable<ChipProps['variant']>;

const COLORS: ChipColor[] = ['default', 'success', 'error', 'info', 'warning'];

const VARIANTS: ChipVariant[] = ['filled', 'outlined'];

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

const meta: Meta<ChipProps> = {
  title: 'Data display/Chip',
  component: Chip,
  tags: ['autodocs'],
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component:
          'A compact element for displaying status, categories, or actions. Extends MUI `ChipProps` (minus `icon`) — all standard props like `sx`, `onClick`, and `component` are also available.',
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
    label: 'Chip',
    variant: 'filled',
    color: 'default',
    size: 'medium',
    clickable: false,
    disabled: false,
    onDelete: undefined,
  },
  argTypes: {
    // Content
    avatar: {
      control: false,
      description: 'Optional leading element, typically an `Avatar` component.',
      table: { category: 'Content' },
    },
    deleteIcon: {
      control: false,
      description:
        'Overrides the default delete icon (`HighlightOff`). Only visible when `onDelete` is provided.',
      table: { category: 'Content' },
    },
    label: {
      control: 'text',
      description: 'The text content of the chip.',
      table: { category: 'Content' },
    },

    // Appearance
    color: {
      control: 'inline-radio',
      options: COLORS,
      description: 'Semantic color of the chip.',
      table: { category: 'Appearance', defaultValue: { summary: "'default'" } },
    },
    size: {
      control: 'inline-radio',
      options: ['small', 'medium'],
      description: 'Size of the chip.',
      table: { category: 'Appearance', defaultValue: { summary: "'medium'" } },
    },
    variant: {
      control: 'inline-radio',
      options: VARIANTS,
      description: 'Visual style of the chip.',
      table: { category: 'Appearance', defaultValue: { summary: "'filled'" } },
    },

    // State
    clickable: {
      control: 'boolean',
      description: 'Makes the chip interactive with hover and focus states.',
      table: { category: 'State', defaultValue: { summary: 'false' } },
    },
    disabled: {
      control: 'boolean',
      description: 'Dims the chip and prevents interaction.',
      table: { category: 'State', defaultValue: { summary: 'false' } },
    },
    onDelete: {
      control: false,
      description:
        'Callback fired when the delete icon is clicked. When provided, a delete icon appears (defaults to `HighlightOff`). Pass a custom `deleteIcon` to override it.',
      table: { category: 'State' },
    },

    // Advanced
    component: {
      control: false,
      description: 'The root element type. Supports polymorphic rendering.',
      table: { category: 'Advanced' },
    },
    sx: {
      control: false,
      description: 'MUI System prop for style overrides. Accepts an `SxProps` object or array.',
      table: { category: 'Advanced' },
    },
  },
  decorators: [
    (Story) => (
      <Box sx={{ p: 2 }}>
        <Story />
      </Box>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

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
  parameters: {
    docs: {
      description: {
        story: 'All available semantic colors shown in both `filled` and `outlined` variants.',
      },
    },
  },
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

export const WithAvatar: Story = {
  parameters: {
    docs: {
      description: {
        story: 'The `avatar` slot accepts any ReactNode, typically an `Avatar` component.',
      },
    },
  },
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
  parameters: {
    docs: {
      description: {
        story:
          'The delete icon only appears when `onDelete` is provided. The default icon is `HighlightOff` — pass a custom `deleteIcon` prop to replace it.',
      },
    },
  },
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
  parameters: {
    docs: {
      description: {
        story:
          'When `clickable` is true, the chip shows hover and focus interactions. Useful for filters or toggleable tags.',
      },
    },
  },
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
