import { type Meta, type StoryObj } from '@storybook/react';
import * as DocBlock from '@storybook/addon-docs/blocks';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import ThemedModePanel from '../../../storybook/themed-mode-panel';
import { makeSeries, timeFormatter } from '../charts.utils';
import StatTile from './stat-tile';
import type { StatTileSize } from './stat-tile.types';

const opsFormatter = (value: number) => `${(value / 1000).toFixed(2)}K ops/s`;
const operations = makeSeries({ seed: 11, base: 1800, spread: 400 });

const signals = [
  {
    label: 'Operations',
    value: '1.84K ops/s',
    data: operations,
    valueFormatter: opsFormatter,
  },
  {
    label: 'Latency',
    value: '4.2 ms',
    data: makeSeries({ seed: 5, base: 4, spread: 1.5 }),
    valueFormatter: (value: number) => `${value.toFixed(1)} ms`,
  },
  {
    label: 'Errors',
    value: '0.3/s',
    data: makeSeries({ seed: 2, base: 0.4, spread: 0.3 }),
    valueFormatter: (value: number) => `${value.toFixed(2)}/s`,
  },
  {
    label: 'Connections',
    value: '312',
    data: makeSeries({ seed: 9, base: 300, spread: 30 }),
    valueFormatter: (value: number) => value.toFixed(0),
  },
];

const meta: Meta<typeof StatTile> = {
  title: 'Data display/Charts/Stat Tile',
  component: StatTile,
  tags: ['autodocs', 'experimental'],
  parameters: {
    layout: 'padded',
    docs: {
      description: {
        component: [
          'A golden-signal tile: what is measured, its current value and a trend of it, such as a cluster’s operations, latency, errors or connections on a Fleet page. Pass the value already formatted with its unit; the tile draws `sparkline` as an area chart unless it says otherwise.',
          '',
          '`size` small fits a dense row of tiles, such as a cluster summary; medium fits an overview page. Small tightens the padding, sets the value one step smaller and draws a shorter trend.',
          '',
          'The tile is a group named by its `label`, and the value is plain text. While `loading`, it shows placeholders and reports itself busy; a sparkline with no points shows `emptyText` instead of an empty chart.',
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
    label: 'Operations',
    value: '1.84K ops/s',
    loading: false,
    size: 'medium',
    sparkline: {
      data: operations,
      valueFormatter: opsFormatter,
      timeFormatter,
      ariaLabel: 'Operations over the last 30 minutes',
    },
  },
  argTypes: {
    label: { control: 'text' },
    value: { control: 'text' },
    sparkline: { control: false },
    emptyText: { control: 'text' },
    size: {
      options: ['small', 'medium'],
      control: { type: 'inline-radio' },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Playground: Story = {
  decorators: [
    (Story) => (
      <Box sx={{ maxWidth: 360 }}>
        <Story />
      </Box>
    ),
  ],
};

const SIZES = ['medium', 'small'] as const satisfies readonly StatTileSize[];

const grid = (size: StatTileSize) => ({
  display: 'grid',
  gridTemplateColumns: `repeat(auto-fill, minmax(${size === 'small' ? 208 : 280}px, 1fr))`,
  gap: size === 'small' ? 1.5 : 2,
});

const SignalTiles = ({ size }: { size: StatTileSize }) => (
  <Box sx={grid(size)}>
    {signals.map(({ label, value, data, valueFormatter }) => (
      <StatTile
        key={label}
        size={size}
        label={label}
        value={value}
        sparkline={{
          data,
          valueFormatter,
          timeFormatter,
          ariaLabel: `${label} over the last 30 minutes`,
        }}
      />
    ))}
  </Box>
);

export const GoldenSignals: Story = {
  name: 'Golden signals',
  parameters: { controls: { disable: true } },
  render: () => <SignalTiles size="medium" />,
};

export const Sizes: Story = {
  parameters: {
    controls: { disable: true },
    docs: {
      description: {
        story:
          'Medium fits an overview page. Small fits a dense row, such as a cluster summary: it tightens the padding, sets the value one step smaller and keeps the trend at 84 × 34, narrowing it only when a long value needs the room.',
      },
    },
  },
  render: () => (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
      {SIZES.map((size) => (
        <Box
          key={size}
          sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}
        >
          <Typography variant="overline">{size}</Typography>
          <SignalTiles size={size} />
        </Box>
      ))}
    </Box>
  ),
};

export const States: Story = {
  name: 'Loading and no data',
  parameters: { controls: { disable: true } },
  render: () => (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
      {SIZES.map((size) => (
        <Box key={size} sx={grid(size)}>
          <StatTile
            size={size}
            label="Operations"
            value="1.84K ops/s"
            sparkline={{ data: operations }}
            loading
          />
          <StatTile
            size={size}
            label="Replication lag"
            value="—"
            sparkline={{ data: [] }}
          />
          <StatTile size={size} label="Uptime" value="41 d" />
        </Box>
      ))}
    </Box>
  ),
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
          <Box
            sx={{
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'flex-start',
              gap: 2,
            }}
          >
            {SIZES.map((size) => (
              <StatTile
                key={size}
                size={size}
                label="Operations"
                value="1.84K ops/s"
                sparkline={{
                  data: operations,
                  valueFormatter: opsFormatter,
                  timeFormatter,
                  ariaLabel: `Operations over the last 30 minutes, ${mode} mode, ${size}`,
                }}
                sx={{ width: size === 'small' ? 208 : 320 }}
              />
            ))}
          </Box>
        </ThemedModePanel>
      ))}
    </Box>
  ),
};
