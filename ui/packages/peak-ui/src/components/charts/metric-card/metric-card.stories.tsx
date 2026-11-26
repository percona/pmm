import { type Meta, type StoryObj } from '@storybook/react';
import * as DocBlock from '@storybook/addon-docs/blocks';
import Box from '@mui/material/Box';
import ThemedModePanel from '../../../storybook/themed-mode-panel';
import { makeSeries, timeFormatter } from '../charts.utils';
import MetricCard from './metric-card';
import type { MetricCardValue } from './metric-card.types';

const queryTime = makeSeries({ seed: 21, base: 1.4, spread: 0.6 });
const msFormatter = (value: number) => `${value.toFixed(2)} ms/s`;
const queryTimeValues: MetricCardValue[] = [
  { label: 'Per second', value: '1.42 ms' },
  { label: 'Sum', value: '2.1 h', note: '34.2%' },
  { label: 'Per query', value: '0.81 ms' },
];

const metrics = [
  {
    title: 'Query time',
    data: queryTime,
    values: queryTimeValues,
    formatter: msFormatter,
  },
  {
    title: 'Rows examined',
    data: makeSeries({ seed: 4, base: 5200, spread: 1600 }),
    values: [
      { label: 'Per second', value: '5.2K' },
      { label: 'Sum', value: '18.7M', note: '61%' },
      { label: 'Per query', value: '412' },
    ],
    formatter: (value: number) => `${(value / 1000).toFixed(1)}K/s`,
  },
  {
    title: 'Query count',
    data: makeSeries({ seed: 8, base: 13, spread: 5 }),
    values: [
      { label: 'Per second', value: '12.6' },
      { label: 'Sum', value: '45.4K', note: '8.9%' },
    ],
    formatter: (value: number) => `${value.toFixed(1)} QPS`,
  },
];

const meta: Meta<typeof MetricCard> = {
  title: 'Data display/Charts/Metric Card',
  component: MetricCard,
  tags: ['autodocs', 'experimental'],
  parameters: {
    layout: 'padded',
    docs: {
      description: {
        component: [
          'A metric summary for a query’s details in Query Analytics: the metric’s name, its trend, and the numbers under it (per second, sum with its share of the total, per query). Pass the numbers already formatted; `sparkline` draws as bars unless it says otherwise.',
          '',
          'The card is a group named by its `title`, and the numbers are a description list, so screen readers read each label with its value. While `loading`, it shows placeholders and reports itself busy; a sparkline with no points shows `emptyText`.',
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
    title: 'Query time',
    values: queryTimeValues,
    loading: false,
    sparkline: {
      data: queryTime,
      valueFormatter: msFormatter,
      timeFormatter,
      ariaLabel: 'Query time per second over the last 30 minutes',
    },
  },
  argTypes: {
    title: { control: 'text' },
    values: { control: 'object' },
    sparkline: { control: false },
    emptyText: { control: 'text' },
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

const grid = {
  display: 'grid',
  gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))',
  gap: 3,
};

export const Metrics: Story = {
  name: 'Query details',
  parameters: { controls: { disable: true } },
  render: () => (
    <Box sx={grid}>
      {metrics.map(({ title, data, values, formatter }) => (
        <MetricCard
          key={title}
          title={title}
          values={values}
          sparkline={{
            data,
            valueFormatter: formatter,
            timeFormatter,
            ariaLabel: `${title} over the last 30 minutes`,
          }}
        />
      ))}
    </Box>
  ),
};

export const States: Story = {
  name: 'Loading and no data',
  parameters: { controls: { disable: true } },
  render: () => (
    <Box sx={grid}>
      <MetricCard
        title="Query time"
        values={queryTimeValues}
        sparkline={{ data: queryTime }}
        loading
      />
      <MetricCard
        title="Lock time"
        values={[
          { label: 'Per second', value: '—' },
          { label: 'Sum', value: '—' },
        ]}
        sparkline={{ data: [] }}
      />
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
          <MetricCard
            title={`Query time (${mode})`}
            values={queryTimeValues}
            sparkline={{
              data: queryTime,
              valueFormatter: msFormatter,
              timeFormatter,
              ariaLabel: `Query time over the last 30 minutes, ${mode} mode`,
            }}
          />
        </ThemedModePanel>
      ))}
    </Box>
  ),
};
