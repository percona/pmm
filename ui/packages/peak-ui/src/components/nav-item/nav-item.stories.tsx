import { type Meta, type StoryObj } from '@storybook/react';
import Box from '@mui/material/Box';
import Chip from '@mui/material/Chip';
import * as DocBlock from '@storybook/blocks';
import HomeOutlined from '@mui/icons-material/HomeOutlined';
import SettingsOutlined from '@mui/icons-material/SettingsOutlined';
import StorageOutlined from '@mui/icons-material/StorageOutlined';
import MonitorHeartOutlined from '@mui/icons-material/MonitorHeartOutlined';
import NavItem from './nav-item';
import type { NavItemProps } from './nav-item.types';

const meta: Meta<NavItemProps> = {
  title: 'Navigation/NavItem',
  component: NavItem,
  tags: ['autodocs'],
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component:
          'A navigation list item. Extends `ListItemButtonProps` — all standard MUI props like `sx`, `onClick`, `disabled`, and `href` are also available.',
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
    text: 'Overview',
    selected: false,
    showDot: false,
    dotColor: 'warning',
  },
  argTypes: {
    // Content
    text: {
      control: 'text',
      description: 'The label for this item.',
      table: { category: 'Content' },
    },
    secondaryText: {
      control: 'text',
      description: 'Auxilliary text when more context is needed for the item.',
      table: { category: 'Content' },
    },
    icon: {
      control: false,
      description:
        'Leading icon rendered before the text. Accepts any ReactNode (typically an MUI icon).',
      table: { category: 'Content' },
    },
    badge: {
      control: false,
      description:
        'Optional trailing slot. Accepts any ReactNode (e.g. a Chip, icon, or custom indicator).',
      table: { category: 'Content' },
    },

    // State
    selected: {
      control: 'boolean',
      description: 'Highlights the item as the current active page.',
      table: { category: 'State', defaultValue: { summary: 'false' } },
    },
    disabled: {
      control: 'boolean',
      description: 'Dims the item and prevents interaction.',
      table: { category: 'State', defaultValue: { summary: 'false' } },
    },

    // Dot
    showDot: {
      control: 'boolean',
      description:
        'When true, a notification dot is rendered over the icon. Only takes effect when `icon` is provided.',
      table: { category: 'Dot indicator', defaultValue: { summary: 'false' } },
    },
    dotColor: {
      control: 'radio',
      options: ['success', 'info', 'warning', 'error'],
      description: 'Color of the notification dot. Only applies when `showDot` is true.',
      table: { category: 'Dot indicator' },
    },

    // Advanced
    component: {
      control: false,
      description:
        'The root element type rendered by ListItemButton. Supports polymorphic rendering (e.g. `a`, `Link`).',
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
      <Box sx={{ width: 320 }}>
        <Story />
      </Box>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Playground: Story = {
  args: {
    text: 'Overview',
    icon: <HomeOutlined />,
  },
};

export const WithBadge: Story = {
  parameters: {
    docs: {
      description: {
        story:
          'The `badge` slot accepts any ReactNode. Here a Chip is used, but it could be an icon, counter, or any custom indicator.',
      },
    },
  },
  args: {
    text: 'Advisors',
    icon: <MonitorHeartOutlined />,
    badge: <Chip size="small" color="success" variant="outlined" label="New feature" />,
  },
};

export const WithoutIcon: Story = {
  parameters: {
    docs: {
      description: {
        story:
          'When no `icon` is provided, the text aligns to the start. Useful for sub-items or text-only navigation.',
      },
    },
  },
  args: {
    text: 'Query Analytics',
  },
};

export const AllStates: Story = {
  parameters: {
    docs: {
      description: {
        story: 'A quick displau of all variants stacked for quick comparison.',
      },
    },
  },
  render: function Render() {
    return (
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
        <NavItem text="Overview" icon={<HomeOutlined />} />
        <NavItem text="Dashboard" selected />
        <NavItem
          text="Services"
          secondaryText="12 active / 2 disabled"
          icon={<StorageOutlined />}
        />
        <NavItem
          text="Advisors"
          icon={<MonitorHeartOutlined />}
          badge={<Chip size="small" color="info" variant="outlined" label="New" />}
        />
        <NavItem text="Settings" icon={<SettingsOutlined />} showDot dotColor="error" />
      </Box>
    );
  },
};
