import { type Meta, type StoryObj } from '@storybook/react';
import * as DocBlock from '@storybook/addon-docs/blocks';
import Box from '@mui/material/Box';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import StatusIcon, {
  type StatusIconProps,
  type StatusIconSeverity,
} from './status-icon';

const SEVERITIES: StatusIconSeverity[] = [
  'success',
  'info',
  'indeterminate',
  'warning',
  'error',
];

const meta = {
  title: 'Feedback/Status Icon',
  component: StatusIcon,
  tags: ['autodocs'],
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component:
          'A small icon, or badge, to signify a system status. It shifts its color style and glyph to convey that status. By choosing a `severity`, you can adapt the messaging tone of the alert. The artwork is locked per severity: to use a different glyph, add a new severity variant. The suggestion is to pair the icon with text next to it (or in the next step of the flow) so the message is as clear as possible.',
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
    severity: 'success',
    size: 'medium',
  },
  argTypes: {
    severity: {
      options: SEVERITIES,
      control: { type: 'inline-radio' },
    },
    size: {
      options: ['medium', 'small'],
      control: { type: 'inline-radio' },
    },
  },
  render: (args) => <StatusIcon {...args} />,
} satisfies Meta<StatusIconProps>;

export default meta;
type Story = StoryObj<StatusIconProps>;

export const Playground: Story = {};

export const Severities: Story = {
  name: 'All severities',
  parameters: { controls: { disable: true } },
  render: () => (
    <Stack direction="row" spacing={4}>
      {SEVERITIES.map((severity) => (
        <Stack key={severity} spacing={1} alignItems="center">
          <StatusIcon severity={severity} />
          <Typography variant="caption" color="text.secondary">
            {severity}
          </Typography>
        </Stack>
      ))}
    </Stack>
  ),
};

export const Sizes: Story = {
  name: 'Sizes',
  parameters: { controls: { disable: true } },
  render: () => (
    <Stack spacing={3}>
      {(['medium', 'small'] as const).map((size) => (
        <Stack key={size} direction="row" spacing={3} alignItems="center">
          <Box sx={{ width: 56 }}>
            <Typography variant="caption" color="text.secondary">
              {size}
            </Typography>
          </Box>
          {SEVERITIES.map((severity) => (
            <StatusIcon key={severity} severity={severity} size={size} />
          ))}
        </Stack>
      ))}
    </Stack>
  ),
};
