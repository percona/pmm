import { type Meta, type StoryObj } from '@storybook/react';
import * as DocBlock from '@storybook/addon-docs/blocks';
import Box from '@mui/material/Box';
import ThemedModePanel from '../../../storybook/themed-mode-panel';
import DistributionBar from './distribution-bar';
import type { DistributionBarSegment } from './distribution-bar.types';

const queryTime: DistributionBarSegment[] = [
  { id: 'planning', label: 'Planning', value: 4.2 },
  { id: 'cpu', label: 'CPU', value: 51.3 },
  { id: 'io', label: 'I/O', value: 31.8 },
  { id: 'lock', label: 'Lock wait', value: 9.6 },
  { id: 'other', label: 'Other', value: 3.1 },
];

const meta: Meta<typeof DistributionBar> = {
  title: 'Data display/Charts/Distribution Bar',
  component: DistributionBar,
  tags: ['autodocs', 'experimental'],
  parameters: {
    layout: 'padded',
    docs: {
      description: {
        component: [
          'A stacked 100% bar that splits a whole into parts, such as where a query spends its time, with a legend under it. Pass raw values in any unit; each part’s share is its value over the sum.',
          '',
          'Parts take the `palette.charts` colors in order unless a segment names its own `color`. A part narrower than 8% drops its in-bar label instead of overflowing; the legend always lists every part with its label and share as text, so the chart never depends on color or hover. The bar itself is hidden from screen readers and the legend is a list, named by `ariaLabel`. `hideLegend` hides it visually only.',
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
    segments: queryTime,
    ariaLabel: 'Query time distribution',
    hideLegend: false,
  },
  argTypes: {
    segments: { control: 'object' },
    valueFormatter: { control: false },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Playground: Story = {};

export const Variants: Story = {
  name: 'All variants',
  render: () => (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
      <DistributionBar
        segments={queryTime}
        ariaLabel="Query time distribution"
      />
      <DistributionBar
        segments={[
          { id: 'read', label: 'Reads', value: 812 },
          { id: 'write', label: 'Writes', value: 188, color: 'chart6' },
        ]}
        valueFormatter={(share, segment) =>
          `${segment.value} (${Math.round(share * 100)}%)`
        }
        ariaLabel="Operations by type"
      />
      <DistributionBar
        segments={queryTime}
        hideLegend
        ariaLabel="Query time distribution"
      />
    </Box>
  ),
  parameters: {
    controls: { disable: true },
    docs: {
      description: {
        story:
          'Default palette; a custom `valueFormatter` and a per-part `color`; and `hideLegend`, for when the legend sits elsewhere on the page (screen readers still get it).',
      },
    },
  },
};

export const Empty: Story = {
  name: 'Nothing measured',
  args: {
    segments: queryTime.map((segment) => ({ ...segment, value: 0 })),
  },
  parameters: {
    docs: {
      description: {
        story:
          'When every part is 0 the bar shows an empty track and the legend reads 0% for each part.',
      },
    },
  },
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
          <DistributionBar
            segments={queryTime}
            ariaLabel={`Query time distribution, ${mode} mode`}
          />
        </ThemedModePanel>
      ))}
    </Box>
  ),
};
