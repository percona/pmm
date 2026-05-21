import { useMemo } from 'react';
import { type Meta, type StoryObj } from '@storybook/react';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import IconButton from '@mui/material/IconButton';
import Typography from '@mui/material/Typography';
import { createTheme, ThemeProvider } from '@mui/material/styles';
import InfoOutlined from '@mui/icons-material/InfoOutlined';
import * as DocBlock from '@storybook/blocks';
import { getThemeOptions } from '../../design';
import Tooltip from './tooltip';
import type { TooltipProps } from './tooltip.types';

type TooltipPlacement = NonNullable<TooltipProps['placement']>;

const PLACEMENTS: TooltipPlacement[] = [
  'top-start',
  'top',
  'top-end',
  'left-start',
  'left',
  'left-end',
  'right-start',
  'right',
  'right-end',
  'bottom-start',
  'bottom',
  'bottom-end',
];

const meta: Meta<TooltipProps> = {
  title: 'Data display/Tooltip',
  component: Tooltip,
  tags: ['autodocs'],
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component:
          'A short, contextual hint shown on hover or focus. Tooltips are always rendered in the inverse color mode: a light app surfaces dark tooltips, a dark app surfaces light tooltips. The inversion is centralized in the `MuiTooltip` theme override, so consumers never need to style tooltips manually. Extends MUI `TooltipProps` directly.',
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
    title: 'Tooltip textual content',
    placement: 'bottom',
    arrow: false,
    disableHoverListener: false,
    disableFocusListener: false,
    disableTouchListener: false,
  },
  argTypes: {
    title: {
      control: 'text',
      description: 'The content shown inside the tooltip. Accepts any ReactNode.',
      table: { category: 'Content' },
    },
    arrow: {
      control: 'boolean',
      description: 'Renders an arrow pointing to the anchor element.',
      table: { category: 'Appearance', defaultValue: { summary: 'false' } },
    },
    placement: {
      control: 'select',
      options: PLACEMENTS,
      description: 'Position of the tooltip relative to the anchor.',
      table: { category: 'Appearance', defaultValue: { summary: "'bottom'" } },
    },
    enterDelay: {
      control: 'number',
      description: 'Delay (ms) before showing the tooltip on hover.',
      table: { category: 'Behavior' },
    },
    leaveDelay: {
      control: 'number',
      description: 'Delay (ms) before hiding the tooltip after the pointer leaves.',
      table: { category: 'Behavior' },
    },
    disableHoverListener: {
      control: 'boolean',
      description: 'Disables showing the tooltip on hover.',
      table: { category: 'Behavior', defaultValue: { summary: 'false' } },
    },
    disableFocusListener: {
      control: 'boolean',
      description: 'Disables showing the tooltip on focus.',
      table: { category: 'Behavior', defaultValue: { summary: 'false' } },
    },
    disableTouchListener: {
      control: 'boolean',
      description: 'Disables showing the tooltip on touch.',
      table: { category: 'Behavior', defaultValue: { summary: 'false' } },
    },
    children: {
      control: false,
      description: 'The anchor element the tooltip is attached to.',
      table: { category: 'Content' },
    },
    sx: {
      control: false,
      description: 'MUI System prop for style overrides. Accepts an `SxProps` object or array.',
      table: { category: 'Advanced' },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

const anchorButton = <Button variant="outlined">Hover me</Button>;

export const Playground: Story = {
  render: function Render({ title, ...args }) {
    return (
      <Tooltip {...args} title={title ?? ''}>
        {anchorButton}
      </Tooltip>
    );
  },
};

export const Placements: Story = {
  parameters: {
    layout: 'padded',
    docs: {
      description: {
        story:
          'All twelve supported placements. Use `placement` to anchor the tooltip relative to its trigger.\n\n**Good to know:** placement is a *preference*, not a guarantee. Tooltips ship with collision detection out of the box — if the requested side does not fit in the viewport, the tooltip flips to the opposite side and shifts along the cross-axis to stay on screen, with the arrow tracking the anchor. No configuration is needed. If a tooltip lives inside a scrollable container that clips it, pass `slotProps.popper.modifiers` to set `preventOverflow.boundary` to `clippingParents`.',
      },
    },
  },
  args: {
    arrow: true,
    open: true,
    disableHoverListener: true,
    disableFocusListener: true,
    disableTouchListener: true,
  },
  render: function Render(args) {
    return (
      <Box
        sx={{
          display: 'grid',
          gridTemplateColumns: 'repeat(3, minmax(160px, 1fr))',
          rowGap: 12,
          columnGap: 8,
          padding: '80px 40px',
          justifyItems: 'center',
        }}
      >
        {PLACEMENTS.map((placement) => (
          <Tooltip key={placement} {...args} placement={placement} title={placement}>
            <Button variant="outlined" size="small" sx={{ minWidth: 128 }}>
              {placement}
            </Button>
          </Tooltip>
        ))}
      </Box>
    );
  },
};

export const WithArrow: Story = {
  args: {
    arrow: true,
    title: 'Tooltip with arrow',
  },
  parameters: {
    docs: {
      description: {
        story:
          'Set `arrow` to render a small caret pointing at the anchor. The arrow uses the same inverted surface color as the tooltip body.',
      },
    },
  },
  render: function Render({ title, ...args }) {
    return (
      <Tooltip {...args} title={title ?? ''}>
        {anchorButton}
      </Tooltip>
    );
  },
};

export const RichContent: Story = {
  args: {
    arrow: true,
    title: (
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1, padding: '4px 2px' }}>
        <Typography variant="subtitle2">Connection failed</Typography>
        <Typography variant="caption">
          The agent on node-03 stopped responding 14 minutes ago. Restart the agent or check the
          network policy.
        </Typography>
      </Box>
    ),
  },
  parameters: {
    docs: {
      description: {
        story:
          'The `title` prop accepts any ReactNode. Compose richer content with typography and layout, but keep it short! Tooltips are not a substitute for popovers or dialogs.',
      },
    },
  },
  render: function Render({ title, ...args }) {
    return (
      <Tooltip {...args} title={title ?? ''}>
        <IconButton aria-label="Connection status">
          <InfoOutlined />
        </IconButton>
      </Tooltip>
    );
  },
};

const themedSurfaceSx = {
  display: 'flex',
  flexDirection: 'column' as const,
  alignItems: 'center',
  justifyContent: 'center',
  gap: 12,
  padding: '32px 24px',
  borderRadius: 8,
  minHeight: 200,
  flex: 1,
};

const ModeShowcase = ({
  mode,
  themeName,
}: {
  mode: 'light' | 'dark';
  themeName: 'base' | 'pmm' | 'sep';
}) => {
  const theme = useMemo(() => createTheme(getThemeOptions(themeName)(mode)), [mode, themeName]);

  return (
    <ThemeProvider theme={theme}>
      <Box
        sx={{
          ...themedSurfaceSx,
          backgroundColor: theme.palette.background.default,
          color: theme.palette.text.primary,
          border: `1px solid ${theme.palette.divider}`,
        }}
      >
        <Typography variant="overline">App in {mode} mode</Typography>
        <Tooltip
          title={`Inverted surface (${mode === 'light' ? 'dark' : 'light'})`}
          arrow
          open
          disableHoverListener
          disableFocusListener
          disableTouchListener
          placement="bottom"
        >
          <Button variant="outlined">Anchor</Button>
        </Tooltip>
        <Box sx={{ height: 4 }} />
      </Box>
    </ThemeProvider>
  );
};

export const ModesShowcase: Story = {
  name: 'Modes (inversion)',
  parameters: {
    layout: 'padded',
    docs: {
      description: {
        story:
          'Side-by-side comparison of how tooltips appear in each app mode. The tooltip surface always uses the opposite mode of the surrounding app, providing strong contrast (AAA in both directions) without any per-tooltip styling.',
      },
    },
  },
  render: function Render() {
    return (
      <Box sx={{ display: 'flex', gap: 2 }}>
        <ModeShowcase mode="light" themeName="base" />
        <ModeShowcase mode="dark" themeName="base" />
      </Box>
    );
  },
};
