import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import { FC, useMemo, useState } from 'react';
import { Messages } from './SessionsTable.messages';
import StopCircleOutlinedIcon from '@mui/icons-material/StopCircleOutlined';
import AddOutlinedIcon from '@mui/icons-material/AddOutlined';
import { Table } from '@percona/ui-lib';
import { boxClasses, paperClasses, Skeleton, Typography } from '@mui/material';
import { SESSIONS_TABLE_COLUMNS } from './SessionsTable.constants';
import { useRealTimeSessions, useStopSessions } from 'hooks/api/useRealTime';
import {
  getAllSessions,
  getServiceIds,
  getSessionRows,
} from './SessionsTable.utils';
import { StopSessionModal } from './modal-stop-session';
import { NewSessionModal } from './modal-new-session';
import StopMultipleSessionsModal from './modal-stop-multiple-sessions/StopMultipleSessionsModal';
import { ModalType, SessionRow } from './SessionsTable.types';
import { enqueueSnackbar } from 'notistack';

const SessionsTable: FC = () => {
  const {
    data: sessions = [],
    isLoading,
    refetch: refetchSessions,
  } = useRealTimeSessions({
    refetchInterval: 5000,
  });
  const rows = getSessionRows(sessions);
  const [modal, setModal] = useState<ModalType>(null);
  const [sessionToBeStopped, setSessionToBeStopped] =
    useState<SessionRow | null>(null);
  const [rowSelection, setRowSelection] = useState<Record<string, boolean>>({});
  const selectedSessions = useMemo(
    () =>
      getAllSessions(rows).filter((session) => rowSelection[session.sessionId]),
    [rowSelection, rows]
  );
  const { mutateAsync: stopSessions } = useStopSessions();

  const closeModal = () => {
    setModal(null);
  };

  const openStopModal = (session: SessionRow) => {
    setSessionToBeStopped(session);
    setModal('stop');
  };

  const handleStopSession = async () => {
    if (!sessionToBeStopped) return;

    const serviceIds = getServiceIds(sessionToBeStopped);
    await stopSessions(serviceIds);

    if (serviceIds.length === 1) {
      enqueueSnackbar(Messages.success.agentStopped, {
        variant: 'success',
      });
    } else {
      enqueueSnackbar(Messages.success.agentsStopped, {
        variant: 'success',
      });
    }

    setSessionToBeStopped(null);

    closeModal();
  };

  const openStopAllModal = () => {
    setModal('stop-all');
  };

  const handleStopAllSessions = async () => {
    const serviceIds = getServiceIds(rows);

    await stopSessions(serviceIds);

    enqueueSnackbar(Messages.success.allAgentsStopped, {
      variant: 'success',
    });

    setRowSelection({});
    closeModal();
  };

  const openNewSessionModal = () => {
    setModal('new-session');
  };

  const openStopSelectedModal = () => {
    setModal('stop-selected');
  };

  const handleStopSelectedSessions = async () => {
    console.log('selectedSessions', selectedSessions);

    if (!selectedSessions.length) return;

    const serviceIds = getServiceIds(selectedSessions);
    await stopSessions(serviceIds);

    if (selectedSessions.length === 1) {
      enqueueSnackbar(Messages.success.agentStopped, {
        variant: 'success',
      });
    } else {
      enqueueSnackbar(Messages.success.agentsStopped, {
        variant: 'success',
      });
    }

    await refetchSessions();
    closeModal();
    setRowSelection({});
  };

  if (isLoading) {
    return <Skeleton variant="rounded" height="100%" />;
  }

  return (
    <Stack
      sx={{
        flex: 1,
        minHeight: 0,
        [`& > .${paperClasses.root}`]: {
          flex: 1,
          display: 'flex',
          flexDirection: 'column',
          minHeight: 0,
          overflow: 'hidden',
        },
      }}
    >
      <Table
        initialState={{
          pagination: {
            pageSize: 25,
            pageIndex: 0,
          },
          columnVisibility: {
            sessionId: true,
          },
          columnOrder: [
            'mrt-row-expand',
            'mrt-row-select',
            ...SESSIONS_TABLE_COLUMNS.map((column) => column.accessorKey || ''),
            'mrt-row-actions',
          ],
        }}
        state={{
          rowSelection,
        }}
        positionToolbarAlertBanner="none"
        getRowId={(row) => row.sessionId}
        noDataMessage={Messages.empty}
        tableName="rta-sessions"
        columns={SESSIONS_TABLE_COLUMNS}
        data={rows}
        enableHiding={false}
        enableGlobalFilter={false}
        enableRowSelection
        onRowSelectionChange={setRowSelection}
        enableStickyHeader
        enableSubRowSelection
        enableExpanding
        enableExpandAll
        enableRowActions
        renderRowActions={({ row }) => (
          <Button
            color="inherit"
            size="small"
            onClick={() => openStopModal(row.original)}
          >
            {Messages.stop}
          </Button>
        )}
        getSubRows={(row) => row.serviceSessions}
        muiTableContainerProps={{
          sx: (theme) => ({
            flex: 1,
            backgroundColor: 'inherit',
            borderTopLeftRadius: theme.shape.borderRadius,
            borderTopRightRadius: theme.shape.borderRadius,
            borderBottom: 0,
          }),
        }}
        muiTableHeadProps={{
          sx: {
            backgroundColor: 'inherit',
          },
        }}
        muiTopToolbarProps={{
          sx: {
            // vertically center the buttons
            [`& > .${boxClasses.root}`]: {
              alignItems: 'center',
              flexDirection: 'row-reverse',
            },
          },
        }}
        renderTopToolbarCustomActions={() => (
          <Stack direction="row" alignItems="center" gap={2}>
            {selectedSessions.length > 0 && (
              <Stack direction="row" alignItems="center" gap={2}>
                <Typography variant="body2">
                  {Messages.selected(selectedSessions.length)}
                </Typography>
                <Button
                  startIcon={<StopCircleOutlinedIcon />}
                  onClick={openStopSelectedModal}
                >
                  {Messages.stopSelected}
                </Button>
              </Stack>
            )}
            {!!sessions.length && (
              <Button
                startIcon={<StopCircleOutlinedIcon />}
                onClick={openStopAllModal}
              >
                {Messages.stopAll}
              </Button>
            )}
            <Button
              startIcon={<AddOutlinedIcon />}
              onClick={openNewSessionModal}
            >
              {Messages.newSession}
            </Button>
          </Stack>
        )}
      />
      <StopSessionModal
        open={modal === 'stop'}
        onClose={closeModal}
        onStopSession={handleStopSession}
      />
      <StopMultipleSessionsModal
        open={modal === 'stop-all' || modal === 'stop-selected'}
        onClose={closeModal}
        onStopSessions={
          modal === 'stop-all'
            ? handleStopAllSessions
            : handleStopSelectedSessions
        }
      />
      <NewSessionModal
        open={modal === 'new-session'}
        onClose={closeModal}
        onSuccess={closeModal}
      />
    </Stack>
  );
};

export default SessionsTable;
