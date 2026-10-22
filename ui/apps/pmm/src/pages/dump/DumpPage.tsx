import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import DownloadIcon from '@mui/icons-material/Download';
import SendIcon from '@mui/icons-material/Send';
import VisibilityOutlinedIcon from '@mui/icons-material/VisibilityOutlined';
import AddOutlinedIcon from '@mui/icons-material/AddOutlined';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Chip from '@mui/material/Chip';
import CircularProgress from '@mui/material/CircularProgress';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import List from '@mui/material/List';
import ListItem from '@mui/material/ListItem';
import ListItemIcon from '@mui/material/ListItemIcon';
import ListItemText from '@mui/material/ListItemText';
import MenuItem from '@mui/material/MenuItem';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { Dialog, DialogTitle, Table } from '@percona/peak-ui';
import { MRT_ColumnDef, MRT_RowSelectionState } from 'material-react-table';
import { FC, useMemo, useState } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import { enqueueSnackbar } from 'notistack';
import { Page } from 'components/page';
import { useUser } from 'contexts/user';
import { useDeleteDumps, useDumps } from 'hooks/api/useDump';
import { useTimezone } from 'hooks/utils/useTimezone';
import { PMM_DUMP_NEW_PATH } from 'lib/constants';
import { Dump, DumpStatus } from 'types/dump.types';
import { OrgRole } from 'types/user.types';
import { DUMP_STATUS } from './DumpPage.constants';
import { Messages } from './DumpPage.messages';
import {
  downloadDumps,
  formatDate,
  formatTimeRange,
  isDumpReady,
} from './DumpPage.utils';
import { DumpLogsDialog } from './components/dump-logs/DumpLogsDialog';
import { SendToSupport } from './components/send-to-support/SendToSupport';

export const DumpPage: FC = () => {
  const { user } = useUser();
  const timezone = useTimezone();
  const { data, isLoading, isError } = useDumps({
    enabled: !!user?.isPMMAdmin,
  });
  const { mutate: remove, isPending: isDeleting } = useDeleteDumps();
  const [rowSelection, setRowSelection] = useState<MRT_RowSelectionState>({});
  const [deleteIds, setDeleteIds] = useState<string[]>([]);
  const [supportIds, setSupportIds] = useState<string[]>([]);
  const [logsDumpId, setLogsDumpId] = useState<string | null>(null);
  const [isLogsOpen, setIsLogsOpen] = useState(false);
  const dumps = data?.dumps ?? [];
  const selectedDumps = dumps.filter(({ dumpId }) => rowSelection[dumpId]);
  const allSelectedReady = selectedDumps.every(isDumpReady);

  const columns = useMemo<MRT_ColumnDef<Dump>[]>(
    () => [
      {
        accessorKey: 'dumpId',
        header: Messages.columns.id,
      },
      {
        accessorKey: 'status',
        header: Messages.columns.status,
        Cell: ({ cell }) => {
          const { label, color } = DUMP_STATUS[cell.getValue<DumpStatus>()];
          return <Chip size="small" label={label} color={color} />;
        },
      },
      {
        accessorKey: 'createdAt',
        header: Messages.columns.created,
        Cell: ({ cell }) => formatDate(cell.getValue<string>(), timezone),
      },
      {
        id: 'timeRange',
        header: Messages.columns.timeRange,
        accessorFn: formatTimeRange,
      },
      {
        accessorKey: 'startTime',
        header: Messages.columns.start,
        Cell: ({ cell }) => formatDate(cell.getValue<string>(), timezone),
      },
      {
        accessorKey: 'endTime',
        header: Messages.columns.end,
        Cell: ({ cell }) => formatDate(cell.getValue<string>(), timezone),
      },
    ],
    [timezone]
  );

  const confirmDelete = () =>
    remove(
      { dumpIds: deleteIds },
      {
        onSuccess: () => {
          enqueueSnackbar(
            deleteIds.length === 1
              ? Messages.delete.success
              : Messages.delete.multipleSuccess,
            { variant: 'success' }
          );
          setDeleteIds([]);
          setRowSelection({});
        },
      }
    );

  return (
    <Page
      title={Messages.title}
      maxWidth="full"
      roles={user?.isPMMAdmin ? undefined : [OrgRole.Admin]}
    >
      <Typography>{Messages.description}</Typography>
      <Stack gap={2} data-testid="pmm-dump-page" mt={1}>
        {isError && <Alert severity="error">{Messages.loadError}</Alert>}
        {isLoading ? (
          <Box display="flex" justifyContent="center" p={4}>
            <CircularProgress aria-label={Messages.loading} />
          </Box>
        ) : (
          <Table
            tableName="pmm-dump"
            columns={columns}
            data={dumps}
            noDataMessage={Messages.empty}
            getRowId={(row) => row.dumpId}
            enableRowSelection
            enableRowActions
            enableExpanding
            enableHiding={false}
            enableGlobalFilter={false}
            positionToolbarAlertBanner="none"
            onRowSelectionChange={setRowSelection}
            state={{ rowSelection }}
            initialState={{
              pagination: { pageIndex: 0, pageSize: 25 },
              columnVisibility: { dumpId: false },
            }}
            renderDetailPanel={({ row }) =>
              row.original.serviceNames.length > 0 ? (
                <Box px={2} py={1}>
                  <Typography fontWeight="bold">
                    {Messages.columns.services}
                  </Typography>
                  <List dense disablePadding>
                    {row.original.serviceNames.map((service) => (
                      <ListItem key={service} disableGutters>
                        {service}
                      </ListItem>
                    ))}
                  </List>
                </Box>
              ) : null
            }
            renderRowActionMenuItems={({ row, closeMenu }) => [
              <MenuItem
                key="download"
                disabled={!isDumpReady(row.original)}
                onClick={() => {
                  downloadDumps([row.original]);
                  closeMenu();
                }}
              >
                <ListItemIcon>
                  <DownloadIcon fontSize="small" />
                </ListItemIcon>
                <ListItemText>{Messages.actions.download}</ListItemText>
              </MenuItem>,
              <MenuItem
                key="support"
                disabled={!isDumpReady(row.original)}
                onClick={() => {
                  setSupportIds([row.original.dumpId]);
                  closeMenu();
                }}
              >
                <ListItemIcon>
                  <SendIcon fontSize="small" />
                </ListItemIcon>
                <ListItemText>{Messages.actions.sendToSupport}</ListItemText>
              </MenuItem>,
              <MenuItem
                key="logs"
                onClick={() => {
                  setLogsDumpId(row.original.dumpId);
                  setIsLogsOpen(true);
                  closeMenu();
                }}
              >
                <ListItemIcon>
                  <VisibilityOutlinedIcon fontSize="small" />
                </ListItemIcon>
                <ListItemText>{Messages.actions.viewLogs}</ListItemText>
              </MenuItem>,
              <MenuItem
                key="delete"
                onClick={() => {
                  setDeleteIds([row.original.dumpId]);
                  closeMenu();
                }}
              >
                <ListItemIcon>
                  <DeleteOutlineIcon fontSize="small" />
                </ListItemIcon>
                <ListItemText>{Messages.actions.delete}</ListItemText>
              </MenuItem>,
            ]}
            renderTopToolbarCustomActions={() => (
              <Stack direction="row" spacing={1} alignItems="center">
                {!!selectedDumps.length && (
                  <>
                    <Typography
                      variant="body2"
                      data-testid="dump-selected-count"
                    >
                      {Messages.selected(selectedDumps.length)}
                    </Typography>
                    <Button
                      startIcon={<SendIcon />}
                      data-testid="dump-sendToSupport"
                      disabled={!allSelectedReady}
                      onClick={() =>
                        setSupportIds(selectedDumps.map(({ dumpId }) => dumpId))
                      }
                    >
                      {Messages.actions.sendToSupport}
                    </Button>
                    <Button
                      startIcon={<DownloadIcon />}
                      data-testid="dump-download-selected"
                      disabled={!allSelectedReady}
                      onClick={() => downloadDumps(selectedDumps)}
                    >
                      {Messages.actions.downloadSelected(selectedDumps.length)}
                    </Button>
                    <Button
                      color="error"
                      startIcon={<DeleteOutlineIcon />}
                      data-testid="dump-delete-selected"
                      onClick={() =>
                        setDeleteIds(selectedDumps.map(({ dumpId }) => dumpId))
                      }
                    >
                      {Messages.actions.deleteSelected(selectedDumps.length)}
                    </Button>
                  </>
                )}
                <Button
                  component={RouterLink}
                  startIcon={<AddOutlinedIcon />}
                  to={PMM_DUMP_NEW_PATH}
                  variant="contained"
                  data-testid="create-dataset"
                >
                  {Messages.createDataset}
                </Button>
              </Stack>
            )}
          />
        )}
      </Stack>
      <Dialog
        open={deleteIds.length > 0}
        onClose={() => setDeleteIds([])}
        loading={isDeleting}
      >
        <DialogTitle onClose={() => setDeleteIds([])}>
          {Messages.delete.title}
        </DialogTitle>
        <DialogContent>
          {deleteIds.length === 1
            ? Messages.delete.single
            : Messages.delete.multiple}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteIds([])} disabled={isDeleting}>
            {Messages.delete.cancel}
          </Button>
          <Button
            color="error"
            variant="contained"
            onClick={confirmDelete}
            disabled={isDeleting}
          >
            {Messages.delete.confirm}
          </Button>
        </DialogActions>
      </Dialog>
      <SendToSupport
        open={supportIds.length > 0}
        dumpIds={supportIds}
        onClose={() => setSupportIds([])}
      />
      <DumpLogsDialog
        key={logsDumpId}
        dumpId={logsDumpId}
        open={isLogsOpen}
        onClose={() => setIsLogsOpen(false)}
      />
    </Page>
  );
};
