import { type Meta, type StoryObj } from '@storybook/react';
import { useState } from 'react';
import * as DocBlock from '@storybook/addon-docs/blocks';
import Box from '@mui/material/Box';
import Card from '@mui/material/Card';
import CardHeader from '@mui/material/CardHeader';
import IconButton from '@mui/material/IconButton';
import InputAdornment from '@mui/material/InputAdornment';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import type { IconButtonProps } from '@mui/material/IconButton';
import CachedOutlined from '@mui/icons-material/CachedOutlined';
import CloseOutlined from '@mui/icons-material/CloseOutlined';
import ContentCopyOutlined from '@mui/icons-material/ContentCopyOutlined';
import InfoOutlined from '@mui/icons-material/InfoOutlined';
import SearchOutlined from '@mui/icons-material/SearchOutlined';
import ShareOutlined from '@mui/icons-material/ShareOutlined';

const meta = {
  title: 'Inputs/Icon Button',
  component: IconButton,
  tags: ['autodocs', 'stable'],
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component: [
          'All three available sizes share a 40 px interactive target. Only the icon size and internal padding changes.',
          '',
          '**Note:** When placed inside a text field adornment, the button keeps its 40 px tap area but occupies only 24 px of layout space (it "bleeds" 8 px on each side).',
        ].join('\n'),
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
  argTypes: {
    size: {
      options: ['small', 'medium', 'large'],
      control: { type: 'inline-radio' },
      description:
        'Button size — changes icon dimensions and internal padding while keeping a 40 px tap target.',
      table: { defaultValue: { summary: "'medium'" } },
    },
    color: {
      options: ['default', 'primary', 'secondary'],
      control: { type: 'inline-radio' },
      description:
        '`default` → `text.primary`, `primary` → `primary.main`, `secondary` → `text.secondary`.',
      table: { defaultValue: { summary: "'default'" } },
    },
    disabled: {
      control: 'boolean',
      description: 'Disables the button and applies the muted palette.',
      table: { defaultValue: { summary: 'false' } },
    },
  },
} satisfies Meta<IconButtonProps>;

export default meta;
type Story = StoryObj<IconButtonProps>;

export const Playground: Story = {
  args: {
    size: 'medium',
    color: 'default',
    disabled: false,
  },
  render: (args) => (
    <IconButton {...args}>
      <CachedOutlined />
    </IconButton>
  ),
};

export const Sizes: Story = {
  parameters: {
    docs: {
      description: {
        story:
          'All three sizes produce a 40 px circle. The icon scales (16 → 20 → 24 px) while padding compensates.',
      },
    },
  },
  render: () => (
    <Stack direction="row" alignItems="center" gap={3}>
      <Stack alignItems="center" gap={1}>
        <IconButton size="small">
          <CachedOutlined />
        </IconButton>
        <Typography variant="caption" color="text.secondary">
          Small
        </Typography>
      </Stack>
      <Stack alignItems="center" gap={1}>
        <IconButton size="medium">
          <CachedOutlined />
        </IconButton>
        <Typography variant="caption" color="text.secondary">
          Medium
        </Typography>
      </Stack>
      <Stack alignItems="center" gap={1}>
        <IconButton size="large">
          <CachedOutlined />
        </IconButton>
        <Typography variant="caption" color="text.secondary">
          Large
        </Typography>
      </Stack>
    </Stack>
  ),
};

export const Colors: Story = {
  parameters: {
    docs: {
      description: {
        story: [
          'The three approved icon colors:',
          '- `default` → `text.primary`',
          '- `primary` → `primary.main`',
          '- `secondary` → `text.secondary`',
        ].join('\n'),
      },
    },
  },
  render: () => (
    <Stack direction="row" gap={3} flexWrap="wrap">
      <Stack alignItems="center" gap={1}>
        <IconButton>
          <CachedOutlined />
        </IconButton>
        <Typography variant="caption" color="text.secondary">
          default
        </Typography>
      </Stack>
      <Stack alignItems="center" gap={1}>
        <IconButton color="primary">
          <CachedOutlined />
        </IconButton>
        <Typography variant="caption" color="text.secondary">
          primary
        </Typography>
      </Stack>
      <Stack alignItems="center" gap={1}>
        <IconButton color="secondary">
          <CachedOutlined />
        </IconButton>
        <Typography variant="caption" color="text.secondary">
          secondary
        </Typography>
      </Stack>
    </Stack>
  ),
};

export const Disabled: Story = {
  parameters: {
    docs: {
      description: {
        story:
          'Disabled buttons are visually muted and ignore pointer events. PMM uses this for prev/next navigation when at boundary rows.',
      },
    },
  },
  render: () => (
    <Stack direction="row" alignItems="center" gap={3}>
      {(['small', 'medium', 'large'] as const).map((s) => (
        <Stack key={s} direction="row" alignItems="center" gap={1}>
          <IconButton size={s}>
            <CachedOutlined />
          </IconButton>
          <IconButton size={s} disabled>
            <CachedOutlined />
          </IconButton>
        </Stack>
      ))}
    </Stack>
  ),
};

export const WithTooltip: Story = {
  name: 'With Tooltip',
  parameters: {
    docs: {
      description: {
        story:
          'The most common PMM pattern: every standalone icon button is wrapped in a `Tooltip` for accessibility. Close, info, share, and refresh actions all follow this layout.',
      },
    },
  },
  render: () => (
    <Stack direction="row" gap={2}>
      <Tooltip title="Close">
        <IconButton>
          <CloseOutlined />
        </IconButton>
      </Tooltip>
      <Tooltip title="More info">
        <IconButton>
          <InfoOutlined />
        </IconButton>
      </Tooltip>
      <Tooltip title="Share">
        <IconButton>
          <ShareOutlined />
        </IconButton>
      </Tooltip>
      <Tooltip title="Copy to clipboard">
        <IconButton>
          <ContentCopyOutlined />
        </IconButton>
      </Tooltip>
    </Stack>
  ),
};

export const InTextField: Story = {
  name: 'In text field',
  parameters: {
    docs: {
      description: {
        story:
          'Inside a text-field adornment the button keeps its 40 px tap area but occupies only 24 px of layout space — it "bleeds" 8 px on each side. This is the pattern MRT uses for the clear-search button.',
      },
    },
  },
  render: function Render() {
    const [value, setValue] = useState('percona');
    return (
      <Stack gap={3} sx={{ width: 280 }}>
        <TextField
          label="With end adornment"
          size="small"
          value={value}
          onChange={(e) => setValue(e.target.value)}
          InputProps={{
            endAdornment: (
              <InputAdornment position="end">
                <Tooltip title="Clear">
                  <span>
                    <IconButton size="small" disabled={!value} onClick={() => setValue('')}>
                      <CloseOutlined />
                    </IconButton>
                  </span>
                </Tooltip>
              </InputAdornment>
            ),
          }}
        />
        <TextField
          label="With start adornment"
          size="small"
          placeholder="Search…"
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <SearchOutlined />
              </InputAdornment>
            ),
          }}
        />
      </Stack>
    );
  },
};

export const DialogClose: Story = {
  name: 'Dialog close',
  parameters: {
    docs: {
      description: {
        story:
          'A common PMM pattern: an absolutely positioned close button in a card or dialog header.',
      },
    },
  },
  render: () => (
    <Card variant="outlined" sx={{ width: 320, position: 'relative' }}>
      <CardHeader
        title="Dialog title"
        action={
          <IconButton aria-label="close">
            <CloseOutlined />
          </IconButton>
        }
      />
      <Box sx={{ px: 2, pb: 2 }}>
        <Typography variant="body2" color="text.secondary">
          The close button sits in the header action slot, matching the PMM modal pattern.
        </Typography>
      </Box>
    </Card>
  ),
};
