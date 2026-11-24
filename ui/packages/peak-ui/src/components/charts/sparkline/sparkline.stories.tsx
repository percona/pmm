import { type Meta, type StoryObj } from '@storybook/react';
import * as DocBlock from '@storybook/addon-docs/blocks';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import ThemedModePanel from '../../../storybook/themed-mode-panel';
import { CHART_SERIES_COLORS } from '../../../design/chart-colors';
import { makeSeries, timeFormatter } from '../charts.utils';
import Sparkline from './sparkline';

const qps = makeSeries({ seed: 7 });
const qpsFormatter = (value: number) => `${value.toFixed(1)} QPS`;

const meta: Meta<typeof Sparkline> = {
  title: 'Data display/Charts/Sparkline',
  component: Sparkline,
  tags: ['autodocs', 'experimental'],
  parameters: {
    layout: 'padded',
    docs: {
      description: {
        component: [
          'An axis-less mini chart for a trend next to a number: in a table row, a metric card or a stat tile. Built on the MIT `SparkLineChart` from `@mui/x-charts`. `bar` suits counts per interval (queries per minute); `area` suits a level over time (latency, connections).',
          '',
          'Hovering shows the value and time of each point. Keyboard users get the same: Tab focuses the chart, the arrow keys step through the points, and each one is announced. Screen readers name the chart with a summary of the latest, lowest and highest value through `valueFormatter`; pass `ariaLabel` to say what the chart shows. Set `decorative` when the same numbers are already printed next to it, such as in a table row: the chart then leaves the tab order and the accessibility tree.',
          '',
          'Marks use `primary` by default, which holds 3:1 contrast on every Peak surface. The `palette.charts` colors hold it in dark mode only, so in light mode keep them for series that are also labelled in text.',
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
    data: qps,
    variant: 'bar',
    height: 48,
    color: 'primary',
    ariaLabel: 'Queries per second over the last 30 minutes',
    valueFormatter: qpsFormatter,
    timeFormatter,
  },
  argTypes: {
    data: { control: false },
    variant: { options: ['bar', 'area'], control: { type: 'inline-radio' } },
    color: {
      options: ['primary', ...CHART_SERIES_COLORS],
      control: { type: 'select' },
    },
    valueFormatter: { control: false },
    timeFormatter: { control: false },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

const narrow: NonNullable<Story['decorators']> = [
  (Story) => (
    <Box sx={{ maxWidth: 320 }}>
      <Story />
    </Box>
  ),
];

export const Playground: Story = { decorators: narrow };

export const Variants: Story = {
  name: 'All variants',
  parameters: { controls: { disable: true } },
  render: () => (
    <Box
      sx={{ display: 'flex', flexDirection: 'column', gap: 3, maxWidth: 320 }}
    >
      <Box>
        <Typography variant="caption" component="p">
          Bar: queries per minute
        </Typography>
        <Sparkline
          data={qps}
          valueFormatter={qpsFormatter}
          timeFormatter={timeFormatter}
          ariaLabel="Queries per second over the last 30 minutes"
        />
      </Box>
      <Box>
        <Typography variant="caption" component="p">
          Area: latency
        </Typography>
        <Sparkline
          variant="area"
          data={makeSeries({ seed: 3, base: 12, spread: 6 })}
          valueFormatter={(value) => `${value.toFixed(1)} ms`}
          timeFormatter={timeFormatter}
          ariaLabel="Latency over the last 30 minutes"
        />
      </Box>
      <Box>
        <Typography variant="caption" component="p">
          Small, decorative: the number is printed next to it
        </Typography>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
          <Typography
            variant="body2"
            sx={{ fontVariantNumeric: 'tabular-nums' }}
          >
            1.2K QPS
          </Typography>
          <Sparkline
            data={qps}
            height={24}
            width={96}
            decorative
            timeFormatter={timeFormatter}
          />
        </Box>
      </Box>
    </Box>
  ),
};

export const Colors: Story = {
  name: 'Chart colors',
  parameters: {
    controls: { disable: true },
    docs: {
      description: {
        story:
          'Every `palette.charts` color, for a sparkline that stands for one series among several. In light mode only `primary` and `chart6` hold 3:1 against the surface, so name the series in text next to the chart.',
      },
    },
  },
  render: () => (
    <Box
      sx={{
        display: 'grid',
        gridTemplateColumns: 'max-content 1fr',
        gap: 1,
        alignItems: 'center',
        maxWidth: 320,
      }}
    >
      {(['primary', ...CHART_SERIES_COLORS] as const).map((color, index) => (
        <Box key={color} sx={{ display: 'contents' }}>
          <Typography variant="caption">{color}</Typography>
          <Sparkline
            variant={index % 2 ? 'area' : 'bar'}
            data={makeSeries({ seed: index + 1, points: 20 })}
            height={32}
            color={color}
            timeFormatter={timeFormatter}
            ariaLabel={`Series ${color}`}
          />
        </Box>
      ))}
    </Box>
  ),
};

export const Flat: Story = {
  name: 'All zero',
  decorators: narrow,
  args: {
    data: qps.map((point) => ({ ...point, value: 0 })),
    ariaLabel: 'No queries in the last 30 minutes',
  },
  parameters: {
    docs: {
      description: {
        story:
          'When every value is 0 the chart draws a dotted baseline, so "nothing happened" reads differently from "no data".',
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
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
            <Sparkline
              data={qps}
              valueFormatter={qpsFormatter}
              timeFormatter={timeFormatter}
              ariaLabel={`Queries per second, ${mode} mode`}
            />
            <Sparkline
              variant="area"
              data={makeSeries({ seed: 3, base: 12, spread: 6 })}
              timeFormatter={timeFormatter}
              ariaLabel={`Latency, ${mode} mode`}
            />
          </Box>
        </ThemedModePanel>
      ))}
    </Box>
  ),
};
