import { type Meta, type StoryObj } from '@storybook/react';
import * as DocBlock from '@storybook/addon-docs/blocks';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import ThemedModePanel from '../../../storybook/themed-mode-panel';
import { CHART_SERIES_COLORS } from '../../../design/chart-colors';
import { formatShare } from '../charts.utils';
import SegmentMeter from './segment-meter';

const meta: Meta<typeof SegmentMeter> = {
  title: 'Data display/Charts/Segment Meter',
  component: SegmentMeter,
  tags: ['autodocs', 'experimental'],
  parameters: {
    layout: 'padded',
    docs: {
      description: {
        component: [
          'A row of dashes that shows a share of the total at a glance, such as one query’s share of the total load in a QAN table row. Any share above 0 fills at least one dash, so a small but real share never looks like nothing.',
          '',
          'Screen readers get a `meter` from 0 to 100 named by `ariaLabel`, read as the percentage. Set `decorative` when the percentage is printed next to the meter. Filled and empty dashes differ in lightness, not only hue.',
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
  args: {
    value: 0.34,
    segments: 5,
    color: 'primary',
    ariaLabel: 'Share of total load',
  },
  argTypes: {
    value: { control: { type: 'range', min: 0, max: 1, step: 0.01 } },
    segments: { control: { type: 'number', min: 1, max: 20 } },
    color: {
      options: ['primary', ...CHART_SERIES_COLORS],
      control: { type: 'select' },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Playground: Story = {};

const SHARES = [0, 0.004, 0.12, 0.34, 0.5, 0.78, 1];

export const Shares: Story = {
  name: 'All shares',
  render: () => (
    <Box
      component="ul"
      sx={{
        listStyle: 'none',
        m: 0,
        p: 0,
        display: 'grid',
        gridTemplateColumns: 'max-content max-content',
        columnGap: 2,
        rowGap: 1,
        alignItems: 'center',
      }}
    >
      {SHARES.map((share) => (
        <Box component="li" key={share} sx={{ display: 'contents' }}>
          <SegmentMeter value={share} decorative />
          <Typography
            variant="caption"
            sx={{ fontVariantNumeric: 'tabular-nums' }}
          >
            {formatShare(share)}
          </Typography>
        </Box>
      ))}
    </Box>
  ),
  parameters: {
    controls: { disable: true },
    docs: {
      description: {
        story:
          'With the percentage printed next to it, the meter is `decorative`, so screen readers read the number once.',
      },
    },
  },
};

export const Empty: Story = {
  name: 'Zero share',
  args: { value: 0 },
};

export const ColorModes: Story = {
  name: 'Light and dark mode',
  parameters: { controls: { disable: true } },
  render: (_args, { globals }) => (
    <Box
      sx={{
        display: 'flex',
        flexDirection: { xs: 'column', md: 'row' },
        gap: 2,
      }}
    >
      {(['light', 'dark'] as const).map((mode) => (
        <ThemedModePanel
          key={mode}
          mode={mode}
          themeName={globals.theme ?? 'base'}
        >
          <Box sx={{ display: 'flex', gap: 2 }}>
            {[0.12, 0.5, 1].map((share) => (
              <SegmentMeter
                key={share}
                value={share}
                ariaLabel={`Share of total load, ${mode} mode`}
              />
            ))}
          </Box>
        </ThemedModePanel>
      ))}
    </Box>
  ),
};
