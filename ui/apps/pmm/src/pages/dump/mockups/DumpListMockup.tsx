import AddOutlinedIcon from '@mui/icons-material/AddOutlined';
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import DownloadIcon from '@mui/icons-material/Download';
import MoreVertIcon from '@mui/icons-material/MoreVert';
import SendIcon from '@mui/icons-material/Send';
import SubjectIcon from '@mui/icons-material/Subject';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import IconButton from '@mui/material/IconButton';
import Link from '@mui/material/Link';
import ListItemIcon from '@mui/material/ListItemIcon';
import ListItemText from '@mui/material/ListItemText';
import Menu from '@mui/material/Menu';
import MenuItem from '@mui/material/MenuItem';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { NothingFoundIllustration, Table, Tooltip } from '@percona/peak-ui';
import type {
  MRT_ColumnDef,
  MRT_RowSelectionState,
} from 'material-react-table';
import { type FC, type MouseEvent, useMemo, useState } from 'react';
import { Page } from 'components/page';
import { type Dump, DumpStatus } from 'types/dump.types';
import {
  CreatedCell,
  DumpStatusCell,
  ServicesCell,
  WindowCell,
} from './DumpCells';
import { DeleteDialogMockup } from './DeleteDialogMockup';
import { DumpLogsMockup } from './DumpLogsMockup';
import { SendToSupportMockup } from './SendToSupportMockup';

export type ListDialog = 'none' | 'support' | 'logs' | 'delete';

interface DumpListMockupProps {
  dumps: Dump[];
  initialSelection?: string[];
  initialDialog?: ListDialog;
  onCreate: () => void;
}

const NOT_READY_HINT = 'Available once the dataset is ready';

const isReady = (dump: Dump) => dump.status === DumpStatus.Success;

const EmptyState: FC<{ onCreate: () => void }> = ({ onCreate }) => (
  <Stack
    alignItems="center"
    textAlign="center"
    gap={2}
    py={8}
    px={2}
    mx="auto"
    maxWidth={480}
    data-testid="dump-empty-state"
  >
    <NothingFoundIllustration
      color="primary"
      aria-hidden
      sx={{ height: 120, width: 120 }}
    />
    <Typography variant="h5" component="h2">
      No datasets yet
    </Typography>
    <Typography color="text.secondary">
      A dataset packs metrics and, optionally, Query Analytics data for the
      services and time range you choose. Percona Support uses it to see what
      your databases were doing when a problem happened.
    </Typography>
    <Button
      variant="contained"
      startIcon={<AddOutlinedIcon />}
      onClick={onCreate}
    >
      Create dataset
    </Button>
  </Stack>
);

export const DumpListMockup: FC<DumpListMockupProps> = ({
  dumps,
  initialSelection = [],
  initialDialog = 'none',
  onCreate,
}) => {
  const [rowSelection, setRowSelection] = useState<MRT_RowSelectionState>(
    Object.fromEntries(initialSelection.map((id) => [id, true]))
  );
  const [dialog, setDialog] = useState<ListDialog>(initialDialog);
  const [target, setTarget] = useState<Dump[]>(() => {
    if (initialDialog === 'logs') {
      return dumps.filter(({ status }) => status === DumpStatus.InProgress);
    }
    return dumps.filter(({ dumpId }) => initialSelection.includes(dumpId));
  });
  const [menu, setMenu] = useState<{ anchor: HTMLElement; dump: Dump } | null>(
    null
  );
  const selected = dumps.filter(({ dumpId }) => rowSelection[dumpId]);
  const notReady = selected.filter((dump) => !isReady(dump)).length;

  const open = (next: ListDialog, items: Dump[]) => {
    setTarget(items);
    setDialog(next);
    setMenu(null);
  };

  const columns = useMemo<MRT_ColumnDef<Dump>[]>(
    () => [
      {
        accessorKey: 'status',
        header: 'Status',
        size: 180,
        Cell: ({ row }) => (
          <DumpStatusCell
            dump={row.original}
            onViewLogs={() => open('logs', [row.original])}
          />
        ),
      },
      {
        accessorKey: 'serviceNames',
        header: 'Services',
        enableSorting: false,
        Cell: ({ row }) => (
          <ServicesCell services={row.original.serviceNames} />
        ),
      },
      {
        id: 'window',
        accessorKey: 'startTime',
        header: 'Data from',
        Cell: ({ row }) => <WindowCell dump={row.original} />,
      },
      {
        accessorKey: 'createdAt',
        header: 'Created',
        size: 140,
        Cell: ({ row }) => <CreatedCell dump={row.original} />,
      },
    ],
    // oxlint-disable-next-line react-hooks/exhaustive-deps
    []
  );

  return (
    <Page title="PMM Dump" maxWidth="full" footer={null}>
      <Stack
        direction={{ xs: 'column', md: 'row' }}
        justifyContent="space-between"
        alignItems={{ xs: 'flex-start', md: 'flex-end' }}
        gap={2}
        mb={2}
      >
        <Typography color="text.secondary" maxWidth={720}>
          Export metrics and Query Analytics data from this PMM Server into an
          archive you can download or upload to Percona Support.{' '}
          <Link href="#">Learn more about PMM Dump</Link>
        </Typography>
        {!!dumps.length && (
          <Button
            variant="contained"
            startIcon={<AddOutlinedIcon />}
            onClick={onCreate}
            sx={{ flexShrink: 0 }}
          >
            Create dataset
          </Button>
        )}
      </Stack>

      {dumps.length === 0 ? (
        <EmptyState onCreate={onCreate} />
      ) : (
        <Table
          tableName="pmm-dump-mockup"
          columns={columns}
          data={dumps}
          getRowId={(row) => row.dumpId}
          enableRowSelection
          enableRowActions
          positionActionsColumn="last"
          enableHiding={false}
          enableGlobalFilter={false}
          enableColumnFilters={false}
          positionToolbarAlertBanner="none"
          onRowSelectionChange={setRowSelection}
          state={{ rowSelection }}
          initialState={{ sorting: [{ id: 'createdAt', desc: true }] }}
          displayColumnDefOptions={{
            'mrt-row-actions': { header: '', size: 96 },
          }}
          renderRowActions={({ row }) => (
            <Stack direction="row" justifyContent="flex-end" width="100%">
              <Tooltip
                title={isReady(row.original) ? 'Download' : NOT_READY_HINT}
              >
                <span>
                  <IconButton
                    aria-label="Download"
                    disabled={!isReady(row.original)}
                    size="small"
                  >
                    <DownloadIcon fontSize="small" />
                  </IconButton>
                </span>
              </Tooltip>
              <IconButton
                aria-label="More actions"
                size="small"
                onClick={(event: MouseEvent<HTMLElement>) =>
                  setMenu({ anchor: event.currentTarget, dump: row.original })
                }
              >
                <MoreVertIcon fontSize="small" />
              </IconButton>
            </Stack>
          )}
          renderTopToolbarCustomActions={() =>
            selected.length ? (
              <Stack direction="row" gap={1} alignItems="center" px={1}>
                <Typography variant="body2" fontWeight={500}>
                  {selected.length} selected
                </Typography>
                <Tooltip
                  title={
                    notReady
                      ? `${notReady} of the selected datasets ${notReady === 1 ? 'is' : 'are'} not ready yet`
                      : ''
                  }
                >
                  <span>
                    <Button
                      size="small"
                      startIcon={<DownloadIcon />}
                      disabled={!!notReady}
                    >
                      Download
                    </Button>
                  </span>
                </Tooltip>
                <Tooltip title={notReady ? NOT_READY_HINT : ''}>
                  <span>
                    <Button
                      size="small"
                      startIcon={<SendIcon />}
                      disabled={!!notReady}
                      onClick={() => open('support', selected)}
                    >
                      Send to Support
                    </Button>
                  </span>
                </Tooltip>
                <Button
                  size="small"
                  color="error"
                  startIcon={<DeleteOutlineIcon />}
                  onClick={() => open('delete', selected)}
                >
                  Delete
                </Button>
              </Stack>
            ) : (
              <Box />
            )
          }
        />
      )}

      <Menu
        anchorEl={menu?.anchor}
        open={!!menu}
        onClose={() => setMenu(null)}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
        transformOrigin={{ vertical: 'top', horizontal: 'right' }}
      >
        <MenuItem
          disabled={!menu || !isReady(menu.dump)}
          onClick={() => menu && open('support', [menu.dump])}
        >
          <ListItemIcon>
            <SendIcon fontSize="small" />
          </ListItemIcon>
          <ListItemText
            primary="Send to Support"
            secondary={menu && !isReady(menu.dump) ? NOT_READY_HINT : undefined}
          />
        </MenuItem>
        <MenuItem onClick={() => menu && open('logs', [menu.dump])}>
          <ListItemIcon>
            <SubjectIcon fontSize="small" />
          </ListItemIcon>
          <ListItemText>View logs</ListItemText>
        </MenuItem>
        <MenuItem
          onClick={() => menu && open('delete', [menu.dump])}
          sx={{ color: 'error.main' }}
        >
          <ListItemIcon>
            <DeleteOutlineIcon fontSize="small" color="error" />
          </ListItemIcon>
          <ListItemText>Delete</ListItemText>
        </MenuItem>
      </Menu>

      <SendToSupportMockup
        open={dialog === 'support'}
        dumps={target}
        onClose={() => setDialog('none')}
      />
      <DumpLogsMockup
        open={dialog === 'logs'}
        dump={target[0]}
        onClose={() => setDialog('none')}
      />
      <DeleteDialogMockup
        open={dialog === 'delete'}
        dumps={target}
        onClose={() => setDialog('none')}
      />
    </Page>
  );
};
