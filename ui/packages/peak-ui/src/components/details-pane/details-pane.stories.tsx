import { useState } from 'react';
import { type Meta, type StoryObj } from '@storybook/react';
import Box from '@mui/material/Box';
import List from '@mui/material/List';
import ListItemButton from '@mui/material/ListItemButton';
import ListItemText from '@mui/material/ListItemText';
import Typography from '@mui/material/Typography';
import { CodeBlock } from '../code';
import LabeledContent from '../labeled-content';
import { useDetailsPaneNavigation } from '../table/useDetailsPaneNavigation';
import DetailsPane from './details-pane';
import type { DetailsPaneProps } from './details-pane.types';

interface Server {
  id: string;
  name: string;
  region: string;
  status: string;
}

const SERVERS: Server[] = [
  {
    id: 'srv-001',
    name: 'db-production-01',
    region: 'us-east-1',
    status: 'Healthy',
  },
  {
    id: 'srv-002',
    name: 'db-staging-02',
    region: 'eu-central-1',
    status: 'Warning',
  },
  {
    id: 'srv-003',
    name: 'db-development-03',
    region: 'ap-southeast-1',
    status: 'Healthy',
  },
];

const serverTabs = (server: Server | undefined): DetailsPaneProps['tabs'] => [
  {
    value: 'details',
    label: 'Details',
    content: server && (
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
        <Typography variant="h5" component="h2">
          {server.name}
        </Typography>
        <LabeledContent label="Region">{server.region}</LabeledContent>
        <LabeledContent label="Status">{server.status}</LabeledContent>
      </Box>
    ),
  },
  {
    value: 'raw-data',
    label: 'Raw data',
    content: server && (
      <CodeBlock
        language="json"
        content={JSON.stringify(server, null, 2)}
        copyable
        wrap
      />
    ),
  },
];

const EXTRA_TABS = ['Metrics', 'Queries', 'Alerts', 'Backups'];

const extraTabs = (server: Server | undefined): DetailsPaneProps['tabs'] =>
  EXTRA_TABS.map((label) => ({
    value: label.toLowerCase(),
    label,
    content: server && <Typography>{`${label} for ${server.name}`}</Typography>,
  }));

type StoryArgs = Pick<DetailsPaneProps, 'labels' | 'defaultTab'> & {
  withNavigation?: boolean;
  manyTabs?: boolean;
};

const ServerListWithPane = ({
  labels,
  defaultTab,
  withNavigation = true,
  manyTabs = false,
}: StoryArgs) => {
  const [selected, setSelected] = useState<Server | undefined>(SERVERS[0]);
  const navigation = useDetailsPaneNavigation<Server>({
    rows: SERVERS,
    selected,
    getRowId: (row) => row.id,
    onSelect: setSelected,
  });

  return (
    <Box
      sx={{
        position: 'relative',
        height: 440,
        overflow: 'hidden',
        border: 1,
        borderColor: 'divider',
        borderRadius: 1,
      }}
    >
      <List>
        {SERVERS.map((server) => (
          <ListItemButton
            key={server.id}
            selected={server.id === selected?.id}
            onClick={() => setSelected(server)}
          >
            <ListItemText primary={server.name} secondary={server.region} />
          </ListItemButton>
        ))}
      </List>
      <DetailsPane
        open={!!selected}
        onClose={() => setSelected(undefined)}
        tabs={
          manyTabs
            ? [...serverTabs(selected), ...extraTabs(selected)]
            : serverTabs(selected)
        }
        defaultTab={defaultTab}
        navigation={withNavigation ? navigation : undefined}
        labels={labels}
        sx={{ top: 64 }}
      />
    </Box>
  );
};

const meta: Meta<typeof DetailsPane> = {
  title: 'Data Display/Details Pane',
  component: DetailsPane,
  tags: ['autodocs', 'experimental'],
  parameters: {
    layout: 'padded',
    docs: {
      description: {
        component: [
          'Bottom sheet that slides up over a table to show one row: tabs on the left, previous, next and close on the right, and the selected tab underneath. It fills its nearest positioned parent, so give the parent `position: relative` and offset the pane with `sx` when it should leave part of the page visible.',
          '',
          'Each tab brings its own `content`; only the selected one is rendered. The pane keeps its own selected tab and goes back to `defaultTab` (or the first tab) when it closes; pass `tab` and `onTabChange` to control it instead.',
          '',
          'Pass the result of `useDetailsPaneNavigation` as `navigation` to step through the rows the user sees; the arrows disable themselves at either end. Without `navigation` only the close button shows. Escape closes the pane while it is open, and closing it returns focus to where it was before the pane opened. Tabs that do not fit scroll, with arrows at either end.',
          '',
          'Name what the buttons move between in `labels` ("Next query", "Next alert"). `previousButtonProps`, `nextButtonProps` and `closeButtonProps` reach the buttons, for example for a `data-testid`, and `tabProps` on a tab does the same for its `Tab`.',
        ].join('\n'),
      },
    },
  },
  args: {
    open: true,
    onClose: () => undefined,
    tabs: [],
  },
  argTypes: {
    open: { control: false },
    tabs: { control: false },
    navigation: { control: false },
  },
  render: (args) => (
    <ServerListWithPane labels={args.labels} defaultTab={args.defaultTab} />
  ),
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const RawDataFirst: Story = {
  name: 'Opens on another tab',
  args: { defaultTab: 'raw-data' },
};

export const NamedButtons: Story = {
  name: 'Buttons named after the item',
  args: {
    labels: {
      previous: 'Previous server',
      next: 'Next server',
      close: 'Close server details',
    },
  },
};

export const WithoutNavigation: Story = {
  name: 'Without navigation',
  parameters: { controls: { disable: true } },
  render: () => <ServerListWithPane withNavigation={false} />,
};

export const ManyTabs: Story = {
  name: 'More tabs than fit',
  parameters: { controls: { disable: true } },
  render: () => (
    <Box sx={{ maxWidth: 420 }}>
      <ServerListWithPane manyTabs />
    </Box>
  ),
};
