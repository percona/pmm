import { type Meta, type StoryObj } from '@storybook/react';
import { useState } from 'react';
import * as DocBlock from '@storybook/addon-docs/blocks';
import Box from '@mui/material/Box';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Button from '@mui/material/Button';
import IconButton from '@mui/material/IconButton';
import Tooltip from '@mui/material/Tooltip';
import Menu from '@mui/material/Menu';
import MenuItem from '@mui/material/MenuItem';
import ListItemIcon from '@mui/material/ListItemIcon';
import ListItemText from '@mui/material/ListItemText';
import Divider from '@mui/material/Divider';
import AddOutlinedIcon from '@mui/icons-material/AddOutlined';
import RefreshOutlinedIcon from '@mui/icons-material/RefreshOutlined';
import DeleteOutlineOutlinedIcon from '@mui/icons-material/DeleteOutlineOutlined';
import InboxOutlinedIcon from '@mui/icons-material/InboxOutlined';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
import MoreVertIcon from '@mui/icons-material/MoreVert';
import Drawer from '@mui/material/Drawer';
import ChevronLeftIcon from '@mui/icons-material/ChevronLeft';
import ChevronRightIcon from '@mui/icons-material/ChevronRight';
import CloseIcon from '@mui/icons-material/Close';
import { type MRT_ColumnDef, type MRT_Row } from 'material-react-table';
import Chip from '../chip';
import Table from './table';
import type { TableProps } from './table.types';
import { useNavigableRows } from './useNavigableRows';
import { useDetailsPaneNavigation } from './useDetailsPaneNavigation';
import { usePerconaTableUrlState } from './usePerconaTableUrlState';

type ServerStatus = 'healthy' | 'warning' | 'error';

type ServerEnvironment = 'production' | 'staging' | 'development';

interface Server {
  id: string;
  name: string;
  status: ServerStatus;
  environment: ServerEnvironment;
  region: string;
  cpu: number;
  memory: number;
  uptimeHours: number;
  processes?: Server[];
}

// Deterministic dataset so visual regression stays stable across runs.
const REGIONS = ['us-east-1', 'us-west-2', 'eu-central-1', 'ap-southeast-1'] as const;
const ENVIRONMENTS: ServerEnvironment[] = ['production', 'staging', 'development'];
const STATUSES: ServerStatus[] = ['healthy', 'healthy', 'healthy', 'warning', 'error'];
const PROCESS_NAMES = ['mysqld', 'mongod', 'postgres', 'redis-server', 'pmm-agent'];

const makeServers = (count: number): Server[] =>
  Array.from({ length: count }, (_, i) => {
    const status = STATUSES[i % STATUSES.length];
    const environment = ENVIRONMENTS[i % ENVIRONMENTS.length];
    const region = REGIONS[i % REGIONS.length];
    return {
      id: `srv-${String(i + 1).padStart(3, '0')}`,
      name: `db-${environment}-${String(i + 1).padStart(2, '0')}`,
      status,
      environment,
      region,
      cpu: (i * 13) % 100,
      memory: (i * 7 + 20) % 100,
      uptimeHours: (i + 1) * 17,
      processes: Array.from({ length: (i % 3) + 1 }, (_, j) => ({
        id: `srv-${i + 1}-proc-${j + 1}`,
        name: PROCESS_NAMES[(i + j) % PROCESS_NAMES.length],
        status: STATUSES[(i + j) % STATUSES.length],
        environment,
        region,
        cpu: ((i + j) * 11) % 100,
        memory: ((i + j) * 5 + 15) % 100,
        uptimeHours: (i + 1) * 17 - j,
        processes: [],
      })),
    };
  });

const SERVERS_30 = makeServers(30);
const SERVERS_8 = makeServers(8);

const STATUS_COLOR: Record<ServerStatus, 'success' | 'warning' | 'error'> = {
  healthy: 'success',
  warning: 'warning',
  error: 'error',
};

const STATUS_RANK: Record<ServerStatus, number> = {
  healthy: 0,
  warning: 1,
  error: 2,
};

const renderStatusCell = ({ row }: { row: MRT_Row<Server> }) => (
  <Chip color={STATUS_COLOR[row.original.status]} label={row.original.status} />
);

const baseColumns: MRT_ColumnDef<Server>[] = [
  {
    header: 'Name',
    accessorKey: 'name',
    // size: 256,
  },
  {
    header: 'Status',
    accessorKey: 'status',
    Cell: renderStatusCell,
    sortingFn: (a, b) => STATUS_RANK[a.original.status] - STATUS_RANK[b.original.status],
    filterVariant: 'select',
    filterSelectOptions: ['healthy', 'warning', 'error'],
  },
  {
    header: 'Environment',
    accessorKey: 'environment',
    filterVariant: 'select',
    filterSelectOptions: ['production', 'staging', 'development'],
  },
  {
    header: 'Region',
    accessorKey: 'region',
  },
  {
    header: 'CPU %',
    accessorKey: 'cpu',
    filterVariant: 'range',
    Cell: ({ cell }) => `${cell.getValue<number>()}%`,
  },
  {
    header: 'Memory %',
    accessorKey: 'memory',
    filterVariant: 'range',
    Cell: ({ cell }) => `${cell.getValue<number>()}%`,
  },
  {
    header: 'Uptime',
    accessorKey: 'uptimeHours',
    Cell: ({ cell }) => `${cell.getValue<number>()}h`,
  },
];

const RowActionsMenu = ({ row }: { row: MRT_Row<Server> }) => {
  const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null);
  const open = Boolean(anchorEl);
  const menuId = `row-actions-${row.id}`;
  const buttonId = `row-actions-trigger-${row.id}`;
  const handleClose = () => setAnchorEl(null);

  return (
    <>
      <Tooltip title="More actions">
        <IconButton
          id={buttonId}
          size="medium"
          aria-label={`Actions for ${row.original.name}`}
          aria-haspopup="menu"
          aria-controls={open ? menuId : undefined}
          aria-expanded={open || undefined}
          onClick={(e) => setAnchorEl(e.currentTarget)}
        >
          <MoreVertIcon />
        </IconButton>
      </Tooltip>
      <Menu
        id={menuId}
        anchorEl={anchorEl}
        open={open}
        onClose={handleClose}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
        transformOrigin={{ vertical: 'top', horizontal: 'right' }}
        slotProps={{ list: { 'aria-labelledby': buttonId, dense: true } }}
      >
        <MenuItem onClick={handleClose}>
          <ListItemIcon>
            <RefreshOutlinedIcon fontSize="small" />
          </ListItemIcon>
          <ListItemText>Refresh</ListItemText>
        </MenuItem>
        <MenuItem onClick={handleClose}>
          <ListItemIcon>
            <InfoOutlinedIcon fontSize="small" />
          </ListItemIcon>
          <ListItemText>View details</ListItemText>
        </MenuItem>
        <Divider />
        <MenuItem onClick={handleClose} sx={{ color: 'error.main' }}>
          <ListItemIcon sx={{ color: 'inherit' }}>
            <DeleteOutlineOutlinedIcon fontSize="small" />
          </ListItemIcon>
          <ListItemText>Remove</ListItemText>
        </MenuItem>
      </Menu>
    </>
  );
};

const EmptyStateIllustration = () => (
  <Stack
    direction="column"
    alignItems="center"
    justifyContent="center"
    gap={1}
    sx={{ width: '100%', py: 6, color: 'text.secondary' }}
  >
    <InboxOutlinedIcon sx={{ fontSize: 48, opacity: 0.4 }} />
    <Typography variant="subtitle1">Nothing to show here yet</Typography>
    <Typography variant="body2">
      Connect a service to start seeing servers in this table.
    </Typography>
    <Button size="small" variant="outlined" startIcon={<AddOutlinedIcon />} sx={{ mt: 1 }}>
      Add service
    </Button>
  </Stack>
);

const meta = {
  title: 'Data display/Table',
  component: Table,
  tags: ['autodocs'],
  parameters: {
    layout: 'padded',
    controls: {
      exclude: [
        'columns',
        'data',
        'getSubRows',
        'renderRowActions',
        'rowHoverAction',
        'noDataAlertProps',
      ],
    },
    docs: {
      description: {
        component: [
          "A themed wrapper around Material React Table [(original docs)](https://www.material-react-table.com/docs/api/table-options). It replaces MUI's `<Table>` for product UI. Switch features on or off depending on the use case:",
          '- **Simple table:** Disable everything, `enableTopToolbar={false}`, `enableColumnActions={false}`, `enableSorting={false}`, etc. See the [Simple Read-only story](?path=/story/data-display-table--simple-read-only).',
          '- **Data table:** Leave defaults on and add/tweak the features you need: filtering, sorting, pagination, row selection, expanding rows, etc.',
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
  argTypes: {
    tableName: {
      control: 'text',
      description:
        '**Peak Design prop.** Used as the `data-testid` and as the localStorage key for column-visibility persistence. Rename with care — changing it loses user-saved preferences.',
      table: { category: 'Data' },
    },
    noDataMessage: {
      control: 'text',
      description:
        '**Peak Design prop.** Message shown when `data` is empty (and no `emptyState` is provided).',
      table: { category: 'Empty state', defaultValue: { summary: "'No data'" } },
    },
    emptyFilterResultsMessage: {
      control: 'text',
      description: '**Peak Design prop.** Message shown when filtering excludes all rows.',
      table: { category: 'Empty state', defaultValue: { summary: "'No data found'" } },
    },
    emptyState: {
      control: false,
      description:
        '**Peak Design prop.** React node that fully replaces the default empty-state alert. See the [Empty state — custom illustration](?path=/story/data-display-table--empty-state-custom) story.',
      table: { category: 'Empty state' },
    },
    enableTopToolbar: {
      control: 'boolean',
      description: 'Show the top toolbar.',
      table: { category: 'Toolbar', defaultValue: { summary: 'true' } },
    },
    enableBottomToolbar: {
      control: 'boolean',
      description:
        '**Peak Design override.** Coupled to `enablePagination` — both auto-enable when `data.length > 10`. In most cases, control pagination instead and this follows.',
      table: { category: 'Toolbar', defaultValue: { summary: 'data.length > 10' } },
    },
    enableGlobalFilter: {
      control: 'boolean',
      description: 'Toolbar search input across all columns.',
      table: { category: 'Toolbar', defaultValue: { summary: 'true' } },
    },
    enableHiding: {
      control: 'boolean',
      description: 'Show/hide columns menu.',
      table: { category: 'Toolbar', defaultValue: { summary: 'true' } },
    },
    enableColumnFilters: {
      control: 'boolean',
      description: 'Per-column filter inputs.',
      table: { category: 'Toolbar', defaultValue: { summary: 'true' } },
    },
    enableColumnActions: {
      control: 'boolean',
      description:
        'Per-column action menu (medium `IconButton`, 20px icon), revealed on header-cell hover or keyboard focus. Opens a dropdown with sort, filter, grouping, pinning, and visibility options provided by MRT.',
      table: { category: 'Toolbar', defaultValue: { summary: 'true' } },
    },
    enableSorting: {
      control: 'boolean',
      description:
        '**Peak Design override.** Auto-disabled when `data` is empty. The sort affordance appears on header-cell hover/focus; the active sort direction stays visible.',
      table: { category: 'Rows', defaultValue: { summary: '!!data.length' } },
    },
    enablePagination: {
      control: 'boolean',
      description:
        '**Peak Design override.** Auto-enabled when `data.length > 10`. When disabled, all rows render without pages and the bottom toolbar hides automatically.',
      table: { category: 'Rows', defaultValue: { summary: 'data.length > 10' } },
    },
    enableStickyHeader: {
      control: 'boolean',
      description:
        'Pin the header while scrolling. Requires a constrained height — pass `muiTableContainerProps={{ sx: { maxHeight: N } }}` to see the effect.',
      table: { category: 'Rows' },
    },
    enableRowSelection: {
      control: 'boolean',
      description:
        'Leading checkbox column. ⚠️ In Storybook, click "Remount" (↻) after toggling — MRT caches display columns on mount.',
      table: { category: 'Rows' },
    },
    enableExpanding: {
      control: 'boolean',
      description:
        'Expandable sub-rows (needs `getSubRows`). ⚠️ In Storybook, click "Remount" (↻) after toggling.',
      table: { category: 'Rows' },
    },
    enableRowActions: {
      control: 'boolean',
      description:
        'Trailing actions column (needs `renderRowActions`). The design system recommends a single kebab `IconButton` that opens a `Menu` — see the [With row actions](?path=/story/data-display-table--with-row-actions) story. ⚠️ In Storybook, click "Remount" (↻) after toggling.',
      table: { category: 'Rows' },
    },
    hideExpandAllIcon: {
      control: 'boolean',
      description:
        '**Peak Design prop.** Hide the expand-all icon in the header. Only meaningful when `enableExpanding` is `true`. Note: MRT offers a native `enableExpandAll={false}` that achieves the same result.',
      table: { category: 'Rows' },
    },
    enableRowHoverAction: {
      control: 'boolean',
      description:
        '**Peak Design prop.** Make whole rows clickable — fires `rowHoverAction(row)` and switches the cursor to a pointer. Use for "click to drill in" patterns.',
      table: { category: 'Rows (Peak Design)', defaultValue: { summary: 'false' } },
    },
    muiTableContainerProps: {
      control: false,
      description:
        'Props forwarded to the `MuiTableContainer` wrapping the `<table>`. This is the visual surface of the table (carries the background color). Use its `sx` to set `maxHeight` for sticky headers or to add a border/border-radius.',
      table: { category: 'Styling' },
    },
  },
} satisfies Meta<TableProps<Server>>;

export default meta;
type Story = StoryObj<TableProps<Server>>;

const splitTableStoryArgs = (args: TableProps<Server>) => {
  const { data, initialState, ...tableArgs } = args;
  return { data, initialState, tableArgs };
};

const WithRowSelectionStory = (args: TableProps<Server>) => {
  const [rowSelection, setRowSelection] = useState<Record<string, boolean>>({});
  const selectedCount = Object.values(rowSelection).filter(Boolean).length;
  return (
    <Table
      {...args}
      state={{ rowSelection }}
      onRowSelectionChange={setRowSelection}
      getRowId={(row) => row.id}
      renderTopToolbarCustomActions={() =>
        selectedCount > 0 ? (
          <Stack direction="row" alignItems="center" gap={1}>
            <Typography variant="body2">{selectedCount} selected</Typography>
            <Button size="small" variant="outlined" startIcon={<DeleteOutlineOutlinedIcon />}>
              Remove
            </Button>
          </Stack>
        ) : null
      }
    />
  );
};

const WithRowHoverActionStory = (args: TableProps<Server>) => {
  const [selected, setSelected] = useState<Server | null>(null);
  return (
    <Stack gap={2}>
      <Table {...args} rowHoverAction={(row) => setSelected(row.original)} />
      {selected && (
        <Box
          sx={{
            p: 2,
            borderRadius: 1,
            backgroundColor: 'background.paper',
            border: 1,
            borderColor: 'divider',
          }}
        >
          <Typography variant="subtitle2">Clicked row</Typography>
          <Typography variant="body2" color="text.secondary">
            {selected.name} — {selected.environment} — {selected.region}
          </Typography>
        </Box>
      )}
    </Stack>
  );
};

const WithDetailsPaneNavigationStory = (args: TableProps<Server>) => {
  const { data: tableData } = args;
  const [selected, setSelected] = useState<Server | undefined>();

  const { navigableRows, tableProps, refresh } = useNavigableRows<Server>({
    data: tableData,
  });

  const { isFirst, isLast, next, previous } = useDetailsPaneNavigation<Server>({
    rows: navigableRows,
    selected,
    getRowId: (row) => row.id,
    onSelect: setSelected,
  });

  return (
    <>
      <Table
        {...args}
        {...tableProps}
        enableRowHoverAction
        rowHoverAction={(row) => {
          refresh();
          setSelected(row.original);
        }}
      />
      <Drawer anchor="right" open={!!selected} onClose={() => setSelected(undefined)}>
        <Stack sx={{ width: 320, p: 2 }} gap={2}>
          <Stack direction="row" alignItems="center" justifyContent="space-between">
            <Stack direction="row" gap={0.5}>
              <Tooltip title="Previous">
                <span>
                  <IconButton
                    size="small"
                    aria-label="Previous"
                    disabled={isFirst}
                    onClick={previous}
                  >
                    <ChevronLeftIcon />
                  </IconButton>
                </span>
              </Tooltip>
              <Tooltip title="Next">
                <span>
                  <IconButton size="small" aria-label="Next" disabled={isLast} onClick={next}>
                    <ChevronRightIcon />
                  </IconButton>
                </span>
              </Tooltip>
            </Stack>
            <IconButton size="small" aria-label="Close" onClick={() => setSelected(undefined)}>
              <CloseIcon />
            </IconButton>
          </Stack>
          {selected && (
            <Box>
              <Typography variant="subtitle1">{selected.name}</Typography>
              <Typography variant="body2" color="text.secondary">
                {selected.environment} — {selected.region} — {selected.status}
              </Typography>
            </Box>
          )}
        </Stack>
      </Drawer>
    </>
  );
};

const WithUrlStateStory = (args: TableProps<Server>) => {
  const { data: tableData, initialState, tableArgs } = splitTableStoryArgs(args);
  const [searchParams, setSearchParams] = useState(() => new URLSearchParams());

  const { tableProps } = usePerconaTableUrlState({
    searchParams,
    setSearchParams,
    initialShowColumnFilters: initialState?.showColumnFilters,
    initialShowGlobalFilter: initialState?.showGlobalFilter,
  });

  return (
    <Stack gap={2}>
      <Typography variant="body2" color="text.secondary" sx={{ fontFamily: 'monospace' }}>
        ?{searchParams.toString() || '(empty)'}
      </Typography>
      <Table {...tableArgs} {...tableProps} data={tableData} />
    </Stack>
  );
};

const WithUrlStateAndDetailsPaneNavigationStory = (args: TableProps<Server>) => {
  const { data: tableData, initialState, tableArgs } = splitTableStoryArgs(args);
  const [searchParams, setSearchParams] = useState(() => new URLSearchParams());
  const [selected, setSelected] = useState<Server | undefined>();

  const { tableState, tableProps: urlTableProps } = usePerconaTableUrlState({
    searchParams,
    setSearchParams,
    initialShowColumnFilters: initialState?.showColumnFilters,
    initialShowGlobalFilter: initialState?.showGlobalFilter,
  });
  const {
    navigableRows,
    tableProps: navTableProps,
    refresh,
  } = useNavigableRows<Server>({
    data: tableData,
    tableState,
  });
  const { isFirst, isLast, next, previous } = useDetailsPaneNavigation<Server>({
    rows: navigableRows,
    selected,
    getRowId: (row) => row.id,
    onSelect: setSelected,
  });

  return (
    <>
      <Typography variant="body2" color="text.secondary" sx={{ fontFamily: 'monospace', mb: 2 }}>
        ?{searchParams.toString() || '(empty)'}
      </Typography>
      <Table
        {...tableArgs}
        {...urlTableProps}
        {...navTableProps}
        data={tableData}
        enableRowHoverAction
        rowHoverAction={(row) => {
          refresh();
          setSelected(row.original);
        }}
      />
      <Drawer anchor="right" open={!!selected} onClose={() => setSelected(undefined)}>
        <Stack sx={{ width: 320, p: 2 }} gap={2}>
          <Stack direction="row" alignItems="center" justifyContent="space-between">
            <Stack direction="row" gap={0.5}>
              <Tooltip title="Previous">
                <span>
                  <IconButton
                    size="small"
                    aria-label="Previous"
                    disabled={isFirst}
                    onClick={previous}
                  >
                    <ChevronLeftIcon />
                  </IconButton>
                </span>
              </Tooltip>
              <Tooltip title="Next">
                <span>
                  <IconButton size="small" aria-label="Next" disabled={isLast} onClick={next}>
                    <ChevronRightIcon />
                  </IconButton>
                </span>
              </Tooltip>
            </Stack>
            <IconButton size="small" aria-label="Close" onClick={() => setSelected(undefined)}>
              <CloseIcon />
            </IconButton>
          </Stack>
          {selected && (
            <Box>
              <Typography variant="subtitle1">{selected.name}</Typography>
              <Typography variant="body2" color="text.secondary">
                {selected.environment} — {selected.region} — {selected.status}
              </Typography>
            </Box>
          )}
        </Stack>
      </Drawer>
    </>
  );
};

export const Playground: Story = {
  args: {
    tableName: 'playground-table',
    columns: baseColumns,
    data: SERVERS_30,
    enableTopToolbar: true,
    enableBottomToolbar: true,
    enableGlobalFilter: false,
    enableHiding: true,
    enableColumnFilters: true,
    enableColumnActions: true,
    enableSorting: true,
    enablePagination: true,
    enableStickyHeader: false,
    enableRowSelection: false,
    enableExpanding: false,
    enableRowActions: false,
    enableRowHoverAction: false,
    hideExpandAllIcon: false,
    noDataMessage: 'No data',
    emptyFilterResultsMessage: 'No data found',
    getSubRows: (row) => row.processes,
    renderRowActions: ({ row }) => <RowActionsMenu row={row} />,
    rowHoverAction: () => {},
  },
};

export const SimpleReadOnly: Story = {
  name: 'Simple read-only',
  parameters: {
    docs: {
      description: {
        story: 'All data-manipulation features disabled. Purely a styled grid of rows and columns.',
      },
    },
  },
  args: {
    tableName: 'simple-table',
    columns: baseColumns.slice(0, 4),
    data: SERVERS_8,
    enableTopToolbar: false,
    enableBottomToolbar: false,
    enableSorting: false,
    enableColumnActions: false,
    enableColumnFilters: false,
    enableHiding: false,
    enablePagination: false,
  },
};

export const WithGlobalSearch: Story = {
  name: 'With global search',
  parameters: {
    docs: {
      description: {
        story: '`enableGlobalFilter` adds a search input that matches across all visible columns.',
      },
    },
  },
  args: {
    tableName: 'with-global-search-table',
    columns: baseColumns,
    data: SERVERS_30,
    enableGlobalFilter: true,
  },
};

export const WithColumnFilters: Story = {
  name: 'With column filters',
  parameters: {
    docs: {
      description: {
        story:
          "Per-column inputs let users filter on individual fields. Columns can declare `filterVariant: 'select'` for enums or `'range'` for numeric ranges.",
      },
    },
  },
  args: {
    tableName: 'with-column-filters-table',
    columns: baseColumns,
    data: SERVERS_30,
    enableColumnFilters: true,
    initialState: { showColumnFilters: true },
  },
};

export const WithRowSelection: Story = {
  name: 'With row selection',
  parameters: {
    docs: {
      description: {
        story:
          'Set `enableRowSelection` to add a leading checkbox column. A neutral alert banner appears showing the selection count. Pair with `renderTopToolbarCustomActions` to show a contextual action bar when rows are selected.',
      },
    },
  },
  render: (args) => <WithRowSelectionStory {...args} />,
  args: {
    tableName: 'with-row-selection-table',
    columns: baseColumns,
    data: SERVERS_30,
    enableRowSelection: true,
  },
};

export const WithExpandingRows: Story = {
  name: 'With expanding rows',
  parameters: {
    docs: {
      description: {
        story:
          '`enableExpanding` with `getSubRows` renders nested sub-rows. Each parent row gets an expand toggle.',
      },
    },
  },
  args: {
    tableName: 'with-expanding-rows-table',
    columns: baseColumns,
    data: SERVERS_30,
    enableExpanding: true,
    getSubRows: (row) => row.processes,
  },
};

export const WithRowActions: Story = {
  name: 'With row actions',
  parameters: {
    docs: {
      description: {
        story: [
          '`enableRowActions` adds a trailing column (scroll to right to see it). Provide `renderRowActions` to define what each row exposes.',
          '',
          '**Recommended pattern:** a single overflow ("kebab") `IconButton` that opens a `Menu`. This keeps rows visually quiet, scales as actions grow, and gives destructive actions clear separation via a `Divider`. Prefer inline icon buttons with appropriate tooltip description only when there is only 1 primary action.',
        ].join('\n'),
      },
    },
  },
  args: {
    tableName: 'with-row-actions-table',
    columns: baseColumns,
    data: SERVERS_30,
    enableRowActions: true,
    renderRowActions: ({ row }) => <RowActionsMenu row={row} />,
  },
};

export const WithCustomToolbarActions: Story = {
  name: 'With custom toolbar actions',
  parameters: {
    docs: {
      description: {
        story:
          '`renderTopToolbarCustomActions` is the slot for page-level controls — refresh, add, bulk operations, etc. The slot sits to the left of the built-in toolbar icons.',
      },
    },
  },
  args: {
    tableName: 'with-custom-toolbar-actions-table',
    columns: baseColumns,
    data: SERVERS_30,
    enableGlobalFilter: true,
    renderTopToolbarCustomActions: () => (
      <Stack direction="row" gap={1}>
        <Button size="small" variant="outlined" startIcon={<RefreshOutlinedIcon />}>
          Refresh
        </Button>
        <Button size="small" variant="contained" startIcon={<AddOutlinedIcon />}>
          Add server
        </Button>
      </Stack>
    ),
  },
};

export const WithRowHoverAction: Story = {
  name: 'With row hover action (Peak Design only)',
  parameters: {
    docs: {
      description: {
        story:
          '`enableRowHoverAction` with `rowHoverAction` makes whole rows clickable. Click a row below to see it in action.',
      },
    },
  },
  render: (args) => <WithRowHoverActionStory {...args} />,
  args: {
    tableName: 'with-row-hover-action-table',
    columns: baseColumns,
    data: SERVERS_30,
    enableRowHoverAction: true,
  },
};

export const StickyHeader: Story = {
  name: 'Sticky header',
  parameters: {
    docs: {
      description: {
        story:
          '`enableStickyHeader` pins the header row when the table scrolls vertically. Constrain the height via `muiTableContainerProps` to see the effect.',
      },
    },
  },
  args: {
    tableName: 'sticky-header-table',
    columns: baseColumns,
    data: SERVERS_30,
    enableStickyHeader: true,
    muiTableContainerProps: { sx: { maxHeight: 360 } },
  },
};

export const WithContainerBorder: Story = {
  name: 'With container border',
  parameters: {
    docs: {
      description: {
        story:
          'Consumers can frame the table surface with a border and rounded corners via `muiTableContainerProps`. This applies to the container that holds the header and body rows — toolbars remain outside.',
      },
    },
  },
  args: {
    tableName: 'with-container-border-table',
    columns: baseColumns,
    data: SERVERS_30,
    muiTableContainerProps: {
      sx: {
        borderRadius: 2,
        border: '1px solid',
        borderColor: 'divider',
      },
    },
  },
};

export const EmptyState: Story = {
  name: 'Empty state — default',
  parameters: {
    docs: {
      description: {
        story:
          'Default empty state when `data` is empty. Customize via `noDataMessage`, or replace entirely with the `emptyState` slot (see next story).',
      },
    },
  },
  args: {
    tableName: 'empty-state-table',
    columns: baseColumns,
    data: [],
    noDataMessage: 'No servers connected yet',
  },
};

export const EmptyStateCustom: Story = {
  name: 'Empty state — custom illustration',
  parameters: {
    docs: {
      description: {
        story:
          'The `emptyState` prop accepts any React node and replaces the default alert entirely.',
      },
    },
  },
  args: {
    tableName: 'empty-state-custom-table',
    columns: baseColumns,
    data: [],
    emptyState: <EmptyStateIllustration />,
  },
};

export const WithDetailsPaneNavigation: Story = {
  name: 'With details pane navigation (prev/next)',
  parameters: {
    docs: {
      description: {
        story: [
          'A common pattern: click a row to open a details pane that has prev/next arrows. The arrows must walk the rows the user actually sees after filtering and sorting, **not** the raw `data` array.',
          '',
          'Two Peak Design hooks make this reusable without hand-wiring the table instance:',
          "- **`useNavigableRows`** owns the table's filter/sort state and returns `navigableRows` (the filtered + sorted rows, across all pages by default) plus `tableProps` to spread onto `<Table>`. It emits `onChange` whenever that list changes, and exposes `refresh()` to recompute right before opening the pane on a row click. Use `scope: 'currentPage'` to navigate only the visible page.",
          '- **`useDetailsPaneNavigation`** turns that list plus the current selection into `next` / `previous` actions and `isFirst` / `isLast` flags. If the selection is filtered out, navigation disables itself instead of jumping to an unrelated row.',
          '- **`usePerconaTableUrlState`** (optional) syncs filter/sort/pagination/global search to URL query params. Pass `searchParams` and `setSearchParams` from your router; spread `tableProps` onto `<Table>` and pass `tableState` into `useNavigableRows` when you need both shareable URLs and details-pane navigation. For other controlled state (e.g. `rowSelection`), pass `additionalState` to the hook or merge with `mergePerconaTableState(tableProps.state, { rowSelection })`.',
          '',
          'Try it: filter by Status or Environment, then open a row and use the arrows — navigation stays within the filtered set.',
        ].join('\n'),
      },
    },
  },
  render: (args) => <WithDetailsPaneNavigationStory {...args} />,
  args: {
    tableName: 'with-details-pane-navigation-table',
    columns: baseColumns,
    data: SERVERS_30,
    enableColumnFilters: true,
    initialState: { showColumnFilters: true },
  },
};

export const WithUrlState: Story = {
  name: 'With URL state (filters, sort, pagination)',
  parameters: {
    docs: {
      description: {
        story: [
          'Demonstrates **`usePerconaTableUrlState`** with an in-memory URL harness (no router dependency).',
          'Filter, sort, search, or paginate the table — the mock query string below updates to match.',
          'Disable filtering per column via `enableColumnFilter: false` on the column definition.',
          'In an app, wire `searchParams` / `setSearchParams` from React Router or your router of choice.',
        ].join('\n'),
      },
    },
  },
  render: (args) => <WithUrlStateStory {...args} />,
  args: {
    tableName: 'with-url-state-table',
    columns: baseColumns,
    data: SERVERS_30,
    enableColumnFilters: true,
    enableGlobalFilter: true,
    initialState: { showColumnFilters: true, showGlobalFilter: true },
  },
};

export const WithUrlStateAndDetailsPaneNavigation: Story = {
  name: 'With URL state + details pane navigation',
  parameters: {
    docs: {
      description: {
        story:
          'Composes **`usePerconaTableUrlState`** and **`useNavigableRows`**: URL owns table state; navigable rows reads the same `tableState` and only supplies `tableInstanceRef` to `<Table>`.',
      },
    },
  },
  render: (args) => <WithUrlStateAndDetailsPaneNavigationStory {...args} />,
  args: {
    tableName: 'with-url-state-details-pane-table',
    columns: baseColumns,
    data: SERVERS_30,
    enableColumnFilters: true,
    enableGlobalFilter: true,
    initialState: { showColumnFilters: true, showGlobalFilter: true },
  },
};

export const EmptyFilterResults: Story = {
  name: 'Empty state — no filter matches',
  parameters: {
    docs: {
      description: {
        story:
          'When filters exclude every row, the table shows `emptyFilterResultsMessage` instead of the no-data message. This story starts with a non-matching filter pre-applied.',
      },
    },
  },
  args: {
    tableName: 'empty-filter-table',
    columns: baseColumns,
    data: SERVERS_30,
    enableGlobalFilter: true,
    emptyFilterResultsMessage: 'No servers match these filters',
    initialState: {
      showGlobalFilter: true,
      globalFilter: 'zzzzz',
    },
  },
};
