import { type Meta, type StoryObj } from '@storybook/react';
import * as DocBlock from '@storybook/addon-docs/blocks';
import Box from '@mui/material/Box';
import ThemedModePanel from '../../../storybook/themed-mode-panel';
import { CHART_SERIES_COLORS } from '../../../design/chart-colors';
import BarList from './bar-list';
import type { BarListItem } from './bar-list.types';

const latency: BarListItem[] = [
  { label: '0-1 ms', value: 0.412 },
  { label: '1-2 ms', value: 0.0635 },
  { label: '2-5 ms', value: 0.281 },
  { label: '5-10 ms', value: 0.164 },
  { label: '10-50 ms', value: 0.071 },
  { label: '50-100 ms', value: 0 },
  { label: '> 100 ms', value: 0.0085 },
];

const meta: Meta<typeof BarList> = {
  title: 'Data display/Charts/Bar List',
  component: BarList,
  tags: ['autodocs', 'experimental'],
  parameters: {
    layout: 'padded',
    docs: {
      description: {
        component: [
          'Labelled horizontal bars, one per row, such as a latency histogram. Bars are drawn relative to the largest value and the value is printed after each bar, so the bars only add the shape.',
          '',
          'It is a description list: screen readers read each label with its value ("1-2 ms, 6.4%") and skip the bars. A row with no value shows `emptyLabel` but reads `emptyText`, so nobody hears "em dash".',
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
    items: latency,
    color: 'primary',
    emptyLabel: '—',
    emptyText: 'No data',
  },
  argTypes: {
    items: { control: 'object' },
    color: {
      options: ['primary', ...CHART_SERIES_COLORS],
      control: { type: 'select' },
    },
    valueFormatter: { control: false },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

const narrow: NonNullable<Story['decorators']> = [
  (Story) => (
    <Box sx={{ maxWidth: 420 }}>
      <Story />
    </Box>
  ),
];

export const Playground: Story = { decorators: narrow };

export const Counts: Story = {
  name: 'Counts with a custom formatter',
  decorators: narrow,
  args: {
    items: [
      { label: 'SELECT', value: 18240 },
      { label: 'INSERT', value: 4210 },
      { label: 'UPDATE', value: 2975 },
      { label: 'DELETE', value: 312 },
    ],
    valueFormatter: (value: number) => value.toLocaleString('en-US'),
    color: 'chart6',
  },
};

export const Empty: Story = {
  name: 'No data',
  decorators: narrow,
  args: { items: latency.map((item) => ({ ...item, value: 0 })) },
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
          <BarList items={latency} />
        </ThemedModePanel>
      ))}
    </Box>
  ),
};
