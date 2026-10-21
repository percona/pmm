import { type Meta, type StoryObj } from '@storybook/react';
import * as DocBlock from '@storybook/addon-docs/blocks';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Card from '@mui/material/Card';
import AddOutlined from '@mui/icons-material/AddOutlined';
import ArrowBack from '@mui/icons-material/ArrowBack';
import ThemedModePanel from '../../../.storybook/themed-mode-panel';
import Table, { type MRT_ColumnDef } from '../table';
import EmptyState, { EMPTY_STATE_ILLUSTRATIONS, EMPTY_STATE_SIZES } from './empty-state';

interface Service {
  name: string;
  type: string;
  environment: string;
}

const serviceColumns: MRT_ColumnDef<Service>[] = [
  { header: 'Name', accessorKey: 'name' },
  { header: 'Type', accessorKey: 'type' },
  { header: 'Environment', accessorKey: 'environment' },
];

const meta: Meta<typeof EmptyState> = {
  title: 'Feedback/Empty State',
  component: EmptyState,
  tags: ['autodocs', 'experimental'],
  parameters: {
    layout: 'padded',
    docs: {
      description: {
        component: [
          'What a place shows when it has content, why it is empty right now, and the way forward, always in that order. Use it for an empty card or section, or a page with nothing on it, instead of composing `Typography` and buttons by hand. `Table` already renders one when `data` is empty; pass one to its `emptyState` slot to add a description, an illustration or actions.',
          '',
          'Low emphasis by design: an empty place on a fresh install is expected, not a problem, so it has no border or color of its own. Inside a card, a table or an outlined section the container already draws the frame; when it stands alone, add a border through `sx`.',
          '',
          '`size` small fits a card, a table or a section; medium fits a page or a whole section. It sets the padding and the illustration size, so the illustration is always the right size for its place. The title is an `h3` by default: set `titleComponent` to fit the page outline, and make it the `h1` only when the empty state is the whole page. The illustration is decorative and hidden from screen readers, so the title and description carry the message.',
          '',
          'Pass one or two MUI `Button`s or `Link`s as `actions`; the first is the way forward. Leave actions out when there is nothing to do: no button beats one that leads nowhere.',
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
    title: 'No services yet',
    description: 'Services show up here once a database is monitored.',
    size: 'medium',
    titleComponent: 'h3',
  },
  argTypes: {
    title: { control: 'text' },
    description: { control: 'text' },
    size: {
      options: ['small', 'medium'],
      control: { type: 'inline-radio' },
    },
    titleComponent: {
      options: ['h2', 'h3', 'h4', 'p'],
      control: { type: 'select' },
    },
    illustration: {
      options: [undefined, ...Object.keys(EMPTY_STATE_ILLUSTRATIONS)],
      control: { type: 'select' },
    },
    actions: { control: false },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const WithAction: Story = {
  args: {
    actions: (
      <Button variant="contained" startIcon={<AddOutlined />}>
        Add service
      </Button>
    ),
  },
};

export const WithTwoActions: Story = {
  args: {
    actions: (
      <>
        <Button variant="contained" startIcon={<AddOutlined />}>
          Add service
        </Button>
        <Button variant="outlined">Install a database</Button>
      </>
    ),
  },
};

export const WithIllustration: Story = {
  parameters: {
    controls: { disable: true },
    docs: {
      description: {
        story: `Name a Peak illustration that fits the moment: \`nothing-found\` for an empty search or filter result, \`radar\` while a scan is still running, \`unknown-error\` when loading failed, \`real-time-database-off\` when a feature is off. Only Peak illustrations are accepted, and \`size\` sets their size (${EMPTY_STATE_SIZES.medium.illustration}px medium, ${EMPTY_STATE_SIZES.small.illustration}px small), so every empty state draws from the same set at the same scale.`,
      },
    },
  },
  render: () => (
    <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', md: '1fr 1fr' }, gap: 2 }}>
      <EmptyState
        illustration="nothing-found"
        title="No matching services"
        description="No service matches “prod-eu”. Try another name, or clear the filters."
        actions={<Button variant="outlined">Clear filters</Button>}
      />
      <EmptyState
        illustration="radar"
        title="Looking for services"
        description="PMM is scanning the node. Services show up here as they are found."
      />
    </Box>
  ),
};

export const SmallInsideCard: Story = {
  name: 'Small, inside a card',
  parameters: { controls: { disable: true } },
  render: () => (
    <Card variant="outlined" sx={{ maxWidth: 400 }}>
      <EmptyState
        size="small"
        title="No backups yet"
        description="Backups start once a service is monitored."
        actions={
          <Button variant="outlined" size="small">
            Set up a backup
          </Button>
        }
      />
    </Card>
  ),
};

export const InsideTable: Story = {
  name: 'Inside a table',
  parameters: {
    controls: { disable: true },
    docs: {
      description: {
        story:
          'An empty `Table` renders a small `EmptyState` titled with `noDataMessage` on its own. Pass one to its `emptyState` slot when the table needs a description, an illustration or a way forward.',
      },
    },
  },
  render: () => (
    <Table<Service>
      tableName="empty-state-inside-table"
      columns={serviceColumns}
      data={[]}
      enableTopToolbar={false}
      emptyState={
        <EmptyState
          size="small"
          title="No services yet"
          description="Connect a service to start seeing it in this table."
          actions={
            <Button variant="outlined" size="small" startIcon={<AddOutlined />}>
              Add service
            </Button>
          }
        />
      }
    />
  ),
};

export const PageLevel: Story = {
  name: 'Page level',
  parameters: {
    controls: { disable: true },
    docs: {
      description: {
        story:
          'When the empty state is the whole page, its title is the page heading: set `titleComponent="h1"`.',
      },
    },
  },
  render: () => (
    <EmptyState
      titleComponent="h1"
      illustration="nothing-found"
      title="Page not found"
      description="The page may have moved, or the link may be wrong."
      actions={
        <>
          <Button variant="contained">Go to home</Button>
          <Button variant="outlined" startIcon={<ArrowBack />}>
            Go back
          </Button>
        </>
      }
    />
  ),
};

const monitoringEmptyState = (
  <EmptyState
    illustration="radar"
    title="Start monitoring"
    description="Add a database service to see its health here, or install a new database on one of your nodes."
    actions={
      <>
        <Button variant="contained" startIcon={<AddOutlined />}>
          Add service
        </Button>
        <Button variant="outlined">Install a database</Button>
      </>
    }
  />
);

export const ColorModes: Story = {
  name: 'Light and dark mode',
  parameters: { controls: { disable: true } },
  render: (_args, { globals }) => (
    <Box sx={{ display: 'flex', flexDirection: { xs: 'column', md: 'row' }, gap: 2 }}>
      <ThemedModePanel mode="light" themeName={globals.theme ?? 'base'}>
        {monitoringEmptyState}
      </ThemedModePanel>
      <ThemedModePanel mode="dark" themeName={globals.theme ?? 'base'}>
        {monitoringEmptyState}
      </ThemedModePanel>
    </Box>
  ),
};
