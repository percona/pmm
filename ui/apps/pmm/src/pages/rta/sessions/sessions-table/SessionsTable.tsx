import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import { FC, useMemo, useState } from 'react';
import { Messages } from './SessionsTable.messages';
import StopCircleOutlinedIcon from '@mui/icons-material/StopCircleOutlined';
import AddOutlinedIcon from '@mui/icons-material/AddOutlined';
import { Table } from '@percona/percona-ui';
import { boxClasses, Skeleton, Typography } from '@mui/material';
import { SESSIONS_TABLE_COLUMNS } from './SessionsTable.constants';
import { useRealtimeSessions, useStopSessions } from 'hooks/api/useRealtime';
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
import { RealtimeTableWrapper } from 'pages/rta/components/rta-table-wrapper';
import { useUser } from 'contexts/user';
import { Navigate } from 'react-router-dom';
import { useTableUrlState } from 'hooks/utils/useTableUrlState';

const SESSIONS_TABLE_URL_STATE_OPTIONS = {
  paramPrefix: 'sessions',
};

const SessionsTable: FC = () => {
  const { user } = useUser();
  const { data: sessions = [], isLoading } = useRealtimeSessions({
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
  const { tableProps } = useTableUrlState({
    ...SESSIONS_TABLE_URL_STATE_OPTIONS,
    additionalState: { rowSelection },
  });

  const closeModal = () => {
    setModal(null);
  };

  const handleStop = async (sessions: SessionRow[], stoppingAll?: boolean) => {
    const serviceIds = getServiceIds(sessions);
    await stopSessions(serviceIds, {
      onSuccess: () => {
        const msg = stoppingAll
          ? Messages.success.allAgentsStopped
          : serviceIds.length === 1
            ? Messages.success.agentStopped
            : Messages.success.agentsStopped;

        enqueueSnackbar(msg, {
          variant: 'success',
        });

        // remove selection from removed item/items
        setRowSelection((selection) => {
          for (const session of getAllSessions(sessions)) {
            delete selection[session.sessionId];
          }
          return { ...selection };
        });
        setSessionToBeStopped(null);
        closeModal();
      },
    });
  };

  const handleStopSession = async () => {
    if (!sessionToBeStopped) return;

    await handleStop([sessionToBeStopped]);
  };

  const handleStopSelectedSessions = async () => {
    if (!selectedSessions.length) return;

    await handleStop(selectedSessions);
  };

  const handleStopAllSessions = async () => {
    await handleStop(rows, true);
  };

  const openStopModal = (session: SessionRow) => {
    setSessionToBeStopped(session);
    setModal('stop');
  };

  const openStopSelectedModal = () => {
    setModal('stop-selected');
  };

  const openStopAllModal = () => {
    setModal('stop-all');
  };

  const openNewSessionModal = () => {
    setModal('new-session');
  };

  if (isLoading) {
    return <Skeleton variant="rounded" height="100%" />;
  }

  if (sessions.length === 0) {
    return <Navigate to="/rta/selection" />;
  }

  return (
    <RealtimeTableWrapper>
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
            ...(user?.isPMMAdmin ? ['mrt-row-actions'] : []),
          ],
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
        displayColumnDefOptions={{
          'mrt-row-select': {
            size: 40,
            muiTableHeadCellProps: { sx: { flex: '0 0 40px !important' } },
            muiTableBodyCellProps: { sx: { flex: '0 0 40px !important' } },
          },
          'mrt-row-expand': {
            size: 40,
          },
        }}
        enableStickyHeader
        enableSubRowSelection
        enableExpanding
        enableExpandAll
        enableRowActions={user?.isPMMAdmin}
        renderRowActions={({ row }) =>
          user?.isPMMAdmin && (
            <Button
              color="inherit"
              size="small"
              data-testid="open-stop-modal"
              onClick={() => openStopModal(row.original as SessionRow)}
            >
              {Messages.stop}
            </Button>
          )
        }
        getSubRows={(row) => row.serviceSessions}
        muiTableContainerProps={{
          sx: {
            flex: 1,
            borderRadius: 2,
            border: '1px solid',
            borderColor: 'divider',
          },
        }}
        muiTopToolbarProps={{
          sx: {
            // vertically center the buttons
            [`& > .${boxClasses.root}`]: {
              alignItems: 'center',
              flexDirection: user?.isPMMAdmin ? 'row-reverse' : undefined,
            },
          },
        }}
        renderTopToolbarCustomActions={() =>
          user?.isPMMAdmin && (
            <Stack direction="row" alignItems="center" gap={2}>
              {selectedSessions.length > 0 && (
                <Stack direction="row" alignItems="center" gap={2}>
                  <Typography variant="body2">
                    {Messages.selected(selectedSessions.length)}
                  </Typography>
                  <Button
                    startIcon={<StopCircleOutlinedIcon />}
                    onClick={openStopSelectedModal}
                    data-testid="open-stop-selected-modal"
                  >
                    {Messages.stopSelected}
                  </Button>
                </Stack>
              )}
              {!!sessions.length && (
                <Button
                  data-testid="open-stop-all-modal"
                  startIcon={<StopCircleOutlinedIcon />}
                  onClick={openStopAllModal}
                >
                  {Messages.stopAll}
                </Button>
              )}
              <Button
                data-testid="open-new-modal"
                startIcon={<AddOutlinedIcon />}
                onClick={openNewSessionModal}
              >
                {Messages.newSession}
              </Button>
            </Stack>
          )
        }
        {...tableProps}
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
    </RealtimeTableWrapper>
  );
};

export default SessionsTable;
