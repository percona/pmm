import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import DownloadIcon from '@mui/icons-material/Download';
import SendIcon from '@mui/icons-material/Send';
import VisibilityOutlinedIcon from '@mui/icons-material/VisibilityOutlined';
import {
  Alert,
  Box,
  Button,
  Chip,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
  MenuItem,
  Stack,
  Typography,
} from '@mui/material';
import { Table } from '@percona/percona-ui';
import { MRT_ColumnDef, MRT_RowSelectionState } from 'material-react-table';
import { FC, useMemo, useState } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import { enqueueSnackbar } from 'notistack';
import { Page } from 'components/page';
import { useUser } from 'contexts/user';
import { useDeleteDumps, useDumps } from 'hooks/api/useDump';
import { Dump, DumpStatus } from 'types/dump.types';
import { OrgRole } from 'types/user.types';
import { STATUS_COLORS } from './DumpPage.constants';
import { Messages } from './DumpPage.messages';
import {
  downloadDumps,
  formatDate,
  formatTimeRange,
  getStatusLabel,
} from './DumpPage.utils';
import { DumpLogsDialog } from './components/dump-logs/DumpLogsDialog';
import { SendToSupport } from './components/send-to-support/SendToSupport';

export const DumpPage: FC = () => {
  const { user } = useUser();
  const { data, isLoading, isError } = useDumps({
    enabled: !!user?.isPMMAdmin,
  });
  const { mutateAsync: remove, isPending: isDeleting } = useDeleteDumps();
  const [rowSelection, setRowSelection] = useState<MRT_RowSelectionState>({});
  const [deleteIds, setDeleteIds] = useState<string[]>([]);
  const [supportIds, setSupportIds] = useState<string[]>([]);
  const [logsDumpId, setLogsDumpId] = useState<string | null>(null);
  const dumps = data?.dumps ?? [];
  const selectedDumps = dumps.filter(({ dumpId }) => rowSelection[dumpId]);

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
          const status = cell.getValue<DumpStatus>();
          return (
            <Chip
              size="small"
              label={getStatusLabel(status)}
              color={STATUS_COLORS[status]}
            />
          );
        },
      },
      {
        accessorKey: 'createdAt',
        header: Messages.columns.created,
        Cell: ({ cell }) => formatDate(cell.getValue<string>()),
      },
      {
        id: 'timeRange',
        header: Messages.columns.timeRange,
        accessorFn: formatTimeRange,
      },
      {
        accessorKey: 'startTime',
        header: Messages.columns.start,
        Cell: ({ cell }) => formatDate(cell.getValue<string>()),
      },
      {
        accessorKey: 'endTime',
        header: Messages.columns.end,
        Cell: ({ cell }) => formatDate(cell.getValue<string>()),
      },
    ],
    []
  );

  const requestDelete = (ids: string[]) => setDeleteIds(ids);

  const confirmDelete = async () => {
    await remove(
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
  };

  const handleDownload = async (items: Dump[]) => {
    try {
      await downloadDumps(items);
    } catch {
      enqueueSnackbar(Messages.downloadError, { variant: 'error' });
    }
  };

  return (
    <Page
      title={Messages.title}
      fullWidth
      surface="paper"
      roles={user?.isPMMAdmin ? undefined : [OrgRole.Admin]}
    >
      <Stack gap={2} data-testid="pmm-dump-page">
        <Stack
          direction={{ xs: 'column', sm: 'row' }}
          justifyContent="space-between"
          alignItems={{ xs: 'stretch', sm: 'center' }}
          gap={1}
        >
          {selectedDumps.length > 0 ? (
            <Stack direction="row" gap={1} flexWrap="wrap">
              <Button
                size="small"
                variant="outlined"
                startIcon={<SendIcon />}
                data-testid="dump-sendToSupport"
                disabled={selectedDumps.some(
                  ({ status }) => status !== DumpStatus.Success
                )}
                onClick={() =>
                  setSupportIds(selectedDumps.map(({ dumpId }) => dumpId))
                }
              >
                {Messages.actions.sendToSupport}
              </Button>
              <Button
                size="small"
                variant="outlined"
                startIcon={<DownloadIcon />}
                data-testid="dump-download-selected"
                disabled={selectedDumps.some(
                  ({ status }) => status !== DumpStatus.Success
                )}
                onClick={() => handleDownload(selectedDumps)}
              >
                {Messages.actions.downloadSelected(selectedDumps.length)}
              </Button>
              <Button
                size="small"
                variant="outlined"
                color="error"
                startIcon={<DeleteOutlineIcon />}
                data-testid="dump-delete-selected"
                onClick={() =>
                  requestDelete(selectedDumps.map(({ dumpId }) => dumpId))
                }
              >
                {Messages.actions.deleteSelected(selectedDumps.length)}
              </Button>
            </Stack>
          ) : (
            <Typography color="text.secondary">
              {Messages.selectDatasets}
            </Typography>
          )}
          <Button
            component={RouterLink}
            to="/pmm-dump/new"
            variant="contained"
            data-testid="create-dataset"
          >
            {Messages.createDataset}
          </Button>
        </Stack>

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
                disabled={row.original.status !== DumpStatus.Success}
                onClick={() => {
                  handleDownload([row.original]);
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
                disabled={row.original.status !== DumpStatus.Success}
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
                  requestDelete([row.original.dumpId]);
                  closeMenu();
                }}
              >
                <ListItemIcon>
                  <DeleteOutlineIcon fontSize="small" />
                </ListItemIcon>
                <ListItemText>{Messages.actions.delete}</ListItemText>
              </MenuItem>,
            ]}
          />
        )}
      </Stack>

      <Dialog open={deleteIds.length > 0} onClose={() => setDeleteIds([])}>
        <DialogTitle>{Messages.delete.title}</DialogTitle>
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
      <DumpLogsDialog dumpId={logsDumpId} onClose={() => setLogsDumpId(null)} />
    </Page>
  );
};
