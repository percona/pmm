import AddOutlinedIcon from '@mui/icons-material/AddOutlined';
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import DownloadIcon from '@mui/icons-material/Download';
import SendIcon from '@mui/icons-material/Send';
import VisibilityOutlinedIcon from '@mui/icons-material/VisibilityOutlined';
import Alert from '@mui/material/Alert';
import Button from '@mui/material/Button';
import IconButton from '@mui/material/IconButton';
import Link from '@mui/material/Link';
import List from '@mui/material/List';
import ListItem from '@mui/material/ListItem';
import ListItemIcon from '@mui/material/ListItemIcon';
import ListItemText from '@mui/material/ListItemText';
import MenuItem from '@mui/material/MenuItem';
import Stack from '@mui/material/Stack';
import type { TableRowProps } from '@mui/material/TableRow';
import Typography from '@mui/material/Typography';
import { Table, type MRT_RowSelectionState } from '@percona/peak-ui';
import { DeleteModal } from 'components/delete-modal';
import { Page } from 'components/page';
import { useUser } from 'contexts/user';
import { useDeleteDumps, useDumps } from 'hooks/api/useDump';
import { useTableUrlState } from 'hooks/utils/useTableUrlState';
import { PMM_DUMP_NEW_PATH } from 'lib/constants';
import { DEFAULT_TABLE_OPTIONS } from 'lib/table.constants';
import { TableMessages } from 'lib/table.messages';
import { MRT_ToggleRowActionMenuButton } from 'material-react-table';
import { enqueueSnackbar } from 'notistack';
import { type FC, useCallback, useMemo, useState } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import type { Dump } from 'types/dump.types';
import { OrgRole } from 'types/user.types';
import { DatasetSummary } from './components/dataset-summary';
import { DumpEmptyState } from './components/dump-empty-state';
import { DumpLogsDialog } from './components/dump-logs/DumpLogsDialog';
import { ReasonTooltip } from './components/reason-tooltip';
import { SendToSupport } from './components/send-to-support/SendToSupport';
import { getDumpColumns } from './DumpPage.columns';
import {
  DUMP_TABLE_NAME,
  DUMP_URL_STATE_OPTIONS,
  PMM_DUMP_DOCS_URL,
} from './DumpPage.constants';
import { Messages } from './DumpPage.messages';
import { downloadDumps, isDumpCreating, isDumpReady } from './DumpPage.utils';

const countWhere = (dumps: Dump[], predicate: (dump: Dump) => boolean) =>
  dumps.filter(predicate).length;

export const DumpPage: FC = () => {
  const { user } = useUser();
  const { data, isLoading, isError } = useDumps({
    enabled: !!user?.isPMMAdmin,
  });
  const { mutateAsync: removeDumps } = useDeleteDumps();
  const [rowSelection, setRowSelection] = useState<MRT_RowSelectionState>({});
  // Dialogs keep their datasets while closing, so their text doesn't change mid-fade
  const [dumpsToDelete, setDumpsToDelete] = useState<Dump[]>([]);
  const [isDeleteOpen, setIsDeleteOpen] = useState(false);
  const [supportDumps, setSupportDumps] = useState<Dump[]>([]);
  const [isSupportOpen, setIsSupportOpen] = useState(false);
  const [logsDump, setLogsDump] = useState<Dump | null>(null);
  const [isLogsOpen, setIsLogsOpen] = useState(false);
  const dumps = useMemo(() => data?.dumps ?? [], [data]);
  const selectedDumps = useMemo(
    () => dumps.filter(({ dumpId }) => rowSelection[dumpId]),
    [dumps, rowSelection]
  );
  const listedSelection = useMemo(
    () => Object.fromEntries(selectedDumps.map(({ dumpId }) => [dumpId, true])),
    [selectedDumps]
  );
  const { tableProps } = useTableUrlState({
    ...DUMP_URL_STATE_OPTIONS,
    additionalState: { rowSelection: listedSelection },
  });
  const hasNoDumps = !isLoading && !dumps.length;
  const notReadyCount = countWhere(selectedDumps, (dump) => !isDumpReady(dump));
  const notReadyReason = notReadyCount
    ? Messages.reasons.bulkNotReady(notReadyCount, selectedDumps.length)
    : undefined;

  const showLogs = useCallback((dump: Dump) => {
    setLogsDump(dump);
    setIsLogsOpen(true);
  }, []);
  const columns = useMemo(() => getDumpColumns(showLogs), [showLogs]);

  const confirmDelete = (items: Dump[]) => {
    setDumpsToDelete(items);
    setIsDeleteOpen(true);
  };

  const sendToSupport = (items: Dump[]) => {
    setSupportDumps(items);
    setIsSupportOpen(true);
  };

  const download = (items: Dump[]) => {
    if (items.length > 1) {
      enqueueSnackbar(Messages.downloading(items.length), { variant: 'info' });
    }
    downloadDumps(items);
  };

  const handleDelete = async () => {
    try {
      await removeDumps({ dumpIds: dumpsToDelete.map(({ dumpId }) => dumpId) });
    } catch {
      // The API interceptor already notified the error
      return;
    }

    enqueueSnackbar(
      dumpsToDelete.length === 1
        ? Messages.delete.success
        : Messages.delete.multipleSuccess,
      { variant: 'success' }
    );
    setIsDeleteOpen(false);
  };

  return (
    <Page
      title={Messages.title}
      maxWidth="full"
      roles={user?.isPMMAdmin ? undefined : [OrgRole.Admin]}
    >
      <Stack gap={2} data-testid="pmm-dump-page">
        <Typography>
          {Messages.description}{' '}
          <Link
            href={PMM_DUMP_DOCS_URL}
            target="_blank"
            rel="noopener noreferrer"
          >
            {Messages.learnMore}
          </Link>
        </Typography>
        {isError && <Alert severity="error">{Messages.loadError}</Alert>}
        {hasNoDumps ? (
          !isError && <DumpEmptyState />
        ) : (
          <Table
            {...DEFAULT_TABLE_OPTIONS}
            {...tableProps}
            state={{
              ...tableProps.state,
              isLoading,
              columnVisibility: { dumpId: false },
            }}
            tableName={DUMP_TABLE_NAME}
            columns={columns}
            data={dumps}
            getRowId={({ dumpId }) => dumpId}
            noDataMessage={Messages.emptyTitle}
            enableRowSelection
            onRowSelectionChange={setRowSelection}
            enableHiding={false}
            enableGlobalFilter={false}
            muiTableBodyRowProps={({ row, isDetailPanel }) =>
              isDetailPanel
                ? {}
                : ({
                    'data-testid': `dump-row-${row.original.dumpId}`,
                  } as TableRowProps)
            }
            enableRowActions
            displayColumnDefOptions={{
              'mrt-row-actions': {
                size: 96,
                // Not renderRowActions: the menu button renders that callback
                // in its place, so nesting the button in it recurses
                Cell: ({ cell, row, staticRowIndex, table }) => {
                  const isReady = isDumpReady(row.original);

                  return (
                    <Stack direction="row" alignItems="center">
                      <ReasonTooltip
                        reason={
                          isReady
                            ? Messages.actions.download
                            : Messages.reasons.notReady
                        }
                      >
                        <IconButton
                          aria-label={Messages.actions.download}
                          disabled={!isReady}
                          onClick={() => download([row.original])}
                          data-testid={`dump-download-${row.original.dumpId}`}
                        >
                          <DownloadIcon />
                        </IconButton>
                      </ReasonTooltip>
                      <MRT_ToggleRowActionMenuButton
                        cell={cell}
                        row={row}
                        staticRowIndex={staticRowIndex}
                        table={table}
                      />
                    </Stack>
                  );
                },
              },
            }}
            renderRowActionMenuItems={({ row: { original }, closeMenu }) => {
              const isReady = isDumpReady(original);

              return [
                <MenuItem
                  key="support"
                  disabled={!isReady}
                  onClick={() => {
                    sendToSupport([original]);
                    closeMenu();
                  }}
                >
                  <ListItemIcon>
                    <SendIcon />
                  </ListItemIcon>
                  <ListItemText
                    primary={Messages.actions.sendToSupport}
                    secondary={isReady ? undefined : Messages.reasons.notReady}
                  />
                </MenuItem>,
                <MenuItem
                  key="logs"
                  onClick={() => {
                    showLogs(original);
                    closeMenu();
                  }}
                >
                  <ListItemIcon>
                    <VisibilityOutlinedIcon />
                  </ListItemIcon>
                  <ListItemText primary={Messages.actions.viewLogs} />
                </MenuItem>,
                <MenuItem
                  key="delete"
                  onClick={() => {
                    confirmDelete([original]);
                    closeMenu();
                  }}
                >
                  <ListItemIcon>
                    <DeleteOutlineIcon color="error" />
                  </ListItemIcon>
                  <ListItemText
                    primary={Messages.actions.delete}
                    slotProps={{ primary: { color: 'error' } }}
                  />
                </MenuItem>,
              ];
            }}
            renderTopToolbarCustomActions={() => (
              <Stack
                direction="row"
                alignItems="center"
                flexWrap="wrap"
                columnGap={2}
                rowGap={1}
              >
                {!!selectedDumps.length && (
                  <>
                    <Typography
                      variant="body2"
                      data-testid="dump-selected-count"
                    >
                      {TableMessages.selected(selectedDumps.length)}
                    </Typography>
                    <ReasonTooltip reason={notReadyReason}>
                      <Button
                        startIcon={<DownloadIcon />}
                        disabled={!!notReadyReason}
                        onClick={() => download(selectedDumps)}
                        data-testid="dump-download-selected"
                      >
                        {Messages.actions.download}
                      </Button>
                    </ReasonTooltip>
                    <ReasonTooltip reason={notReadyReason}>
                      <Button
                        startIcon={<SendIcon />}
                        disabled={!!notReadyReason}
                        onClick={() => sendToSupport(selectedDumps)}
                        data-testid="dump-sendToSupport"
                      >
                        {Messages.actions.sendToSupport}
                      </Button>
                    </ReasonTooltip>
                    <Button
                      color="error"
                      startIcon={<DeleteOutlineIcon />}
                      onClick={() => confirmDelete(selectedDumps)}
                      data-testid="dump-delete-selected"
                    >
                      {Messages.actions.delete}
                    </Button>
                  </>
                )}
                <Button
                  component={RouterLink}
                  to={PMM_DUMP_NEW_PATH}
                  variant="contained"
                  startIcon={<AddOutlinedIcon />}
                  data-testid="create-dataset"
                >
                  {Messages.createDataset}
                </Button>
              </Stack>
            )}
          />
        )}
      </Stack>
      <DeleteModal
        testId="dump-delete-modal"
        open={isDeleteOpen}
        title={Messages.delete.title(dumpsToDelete.length)}
        message={Messages.delete.message}
        confirmLabel={Messages.delete.confirm(dumpsToDelete.length)}
        warning={
          dumpsToDelete.some(isDumpCreating)
            ? Messages.delete.creatingWarning
            : undefined
        }
        onConfirm={handleDelete}
        onClose={() => setIsDeleteOpen(false)}
      >
        <List dense>
          {dumpsToDelete.map((dump) => (
            <ListItem key={dump.dumpId} disableGutters>
              <DatasetSummary dump={dump} />
            </ListItem>
          ))}
        </List>
      </DeleteModal>
      <SendToSupport
        open={isSupportOpen}
        dumps={supportDumps}
        onClose={() => setIsSupportOpen(false)}
      />
      <DumpLogsDialog
        key={logsDump?.dumpId}
        dump={logsDump}
        open={isLogsOpen}
        onClose={() => setIsLogsOpen(false)}
      />
    </Page>
  );
};
