import { type Meta, type StoryObj } from '@storybook/react';
import { useMemo } from 'react';
import * as DocBlock from '@storybook/addon-docs/blocks';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Card from '@mui/material/Card';
import Typography from '@mui/material/Typography';
import { createTheme, ThemeProvider } from '@mui/material/styles';
import AddOutlined from '@mui/icons-material/AddOutlined';
import ArrowBack from '@mui/icons-material/ArrowBack';
import { getThemeOptions } from '../../design';
import { NothingFoundIllustration, RadarIllustration } from '../illustrations';
import Table, { type MRT_ColumnDef } from '../table';
import EmptyState from './empty-state';

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
    controls: { exclude: ['illustration', 'actions'] },
    docs: {
      description: {
        component: [
          'What a place shows when it has content, why it is empty right now, and the way forward, always in that order. Use it for an empty table (through the `emptyState` slot of `Table`), an empty card or section, or a page with nothing on it, instead of composing `Typography` and buttons by hand.',
          '',
          'Low emphasis by design: an empty place on a fresh install is expected, not a problem, so it has no border or color of its own. Inside a card, a table or an outlined section the container already draws the frame; when it stands alone, add a border through `sx`.',
          '',
          '`size` small fits a card, a table or a section; medium fits a page or a whole section. The title is an `h3` by default: set `titleComponent` to fit the page outline, and make it the `h1` only when the empty state is the whole page. The illustration is decorative and hidden from screen readers, so the title and description carry the message.',
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
        story:
          'Pass a Peak illustration that names the moment: Nothing found for an empty search or filter result, Radar while a scan is still running. It is sized by `size` (64px medium, 48px small).',
      },
    },
  },
  render: () => (
    <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', md: '1fr 1fr' }, gap: 2 }}>
      <EmptyState
        illustration={<NothingFoundIllustration />}
        title="No matching services"
        description="No service matches “prod-eu”. Try another name, or clear the filters."
        actions={<Button variant="outlined">Clear filters</Button>}
      />
      <EmptyState
        illustration={<RadarIllustration />}
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
        story: "Pass it to `Table`'s `emptyState` slot, at small size.",
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
      illustration={<NothingFoundIllustration />}
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

const ModePanel = ({ mode, themeName }: { mode: 'light' | 'dark'; themeName: string }) => {
  const theme = useMemo(() => createTheme(getThemeOptions(themeName)(mode)), [mode, themeName]);

  return (
    <ThemeProvider theme={theme}>
      <Box
        sx={{
          flex: 1,
          bgcolor: 'background.paper',
          color: 'text.primary',
          border: 1,
          borderColor: 'divider',
          borderRadius: 1,
        }}
      >
        <Typography variant="overline" component="p" sx={{ px: 2, pt: 2 }}>
          {mode === 'light' ? 'Light mode' : 'Dark mode'}
        </Typography>
        <EmptyState
          illustration={<RadarIllustration />}
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
      </Box>
    </ThemeProvider>
  );
};

export const ColorModes: Story = {
  name: 'Light and dark mode',
  parameters: { controls: { disable: true } },
  render: (_args, { globals }) => (
    <Box sx={{ display: 'flex', flexDirection: { xs: 'column', md: 'row' }, gap: 2 }}>
      <ModePanel mode="light" themeName={globals.theme ?? 'base'} />
      <ModePanel mode="dark" themeName={globals.theme ?? 'base'} />
    </Box>
  ),
};
