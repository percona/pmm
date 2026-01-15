import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import { FC, useMemo, useState } from 'react';
import { Messages } from './SessionsTable.messages';
import StopCircleOutlinedIcon from '@mui/icons-material/StopCircleOutlined';
import AddOutlinedIcon from '@mui/icons-material/AddOutlined';
import { Table } from '@percona/ui-lib';
import { boxClasses, paperClasses, Skeleton } from '@mui/material';
import { SESSIONS_TABLE_COLUMNS } from './SessionsTable.constants';
import {
  useChangeRealTimeAgent,
  useRealTimeAgents,
} from 'hooks/api/useRealTime';
import { getSessions } from './SessionsTable.utils';
import { RealTimeSession } from 'types/rta.types';
import { StopSessionModal } from './modal-stop-session';
import { NewSessionModal } from './modal-new-session';
import StopMultipleSessionsModal from './modal-stop-multiple-sessions/StopMultipleSessionsModal';
import { ModalType } from './SessionsTable.types';
import { enqueueSnackbar } from 'notistack';

const SessionsTable: FC = () => {
  const {
    data: agents = [],
    isLoading,
    refetch: refetchAgents,
  } = useRealTimeAgents({
    refetchInterval: 5000,
  });
  const [modal, setModal] = useState<ModalType>(null);
  const sessions = getSessions(agents);
  const [sessionToBeStopped, setSessionToBeStopped] =
    useState<RealTimeSession | null>(null);
  const [rowSelection, setRowSelection] = useState<Record<string, boolean>>({});
  const selectedSessions = useMemo(
    () => sessions.filter((session) => rowSelection[session.sessionId]),
    [rowSelection, sessions]
  );
  const { mutateAsync: changeRealTimeAgent } = useChangeRealTimeAgent();

  const closeModal = () => {
    setModal(null);
  };

  const openStopModal = (session: RealTimeSession) => {
    setSessionToBeStopped(session);
    setModal('stop');
  };

  const handleStopSession = async () => {
    if (sessionToBeStopped) {
      await Promise.all(
        sessionToBeStopped.agents.map((agent) =>
          changeRealTimeAgent({
            serviceId: agent.serviceId,
            enable: false,
          })
        )
      );

      if (sessionToBeStopped.agents.length === 1) {
        enqueueSnackbar(Messages.success.agentStopped, {
          variant: 'success',
        });
      } else {
        enqueueSnackbar(Messages.success.agentsStopped, {
          variant: 'success',
        });
      }
    }

    await refetchAgents();
    closeModal();
  };

  const openStopAllModal = () => {
    setModal('stop-all');
  };

  const handleStopAllSessions = async () => {
    await Promise.all(
      sessions.map((session) =>
        Promise.all(
          session.agents.map((agent) =>
            changeRealTimeAgent({
              serviceId: agent.serviceId,
              enable: false,
            })
          )
        )
      )
    );

    enqueueSnackbar(Messages.success.allAgentsStopped, {
      variant: 'success',
    });

    await refetchAgents();

    setRowSelection({});
    closeModal();
  };

  const openNewSessionModal = () => {
    setModal('new-session');
  };

  const handleCreateSessions = async (serviceIds: string[]) => {
    await Promise.all(
      serviceIds.map((serviceId) =>
        changeRealTimeAgent({
          serviceId,
          enable: true,
        })
      )
    );
    await refetchAgents();
    closeModal();
  };

  const openStopSelectedModal = () => {
    setModal('stop-selected');
  };

  const handleStopSelectedSessions = async () => {
    await Promise.all(
      selectedSessions.map((session) =>
        Promise.all(
          session.agents.map((agent) =>
            changeRealTimeAgent({
              serviceId: agent.serviceId,
              enable: false,
            })
          )
        )
      )
    );

    if (selectedSessions.length === 1) {
      enqueueSnackbar(Messages.success.agentStopped, {
        variant: 'success',
      });
    } else {
      enqueueSnackbar(Messages.success.agentsStopped, {
        variant: 'success',
      });
    }

    await refetchAgents();
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
            sessionId: false,
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
        getRowId={(row) => row.sessionId}
        noDataMessage={Messages.empty}
        tableName="rta-sessions"
        columns={SESSIONS_TABLE_COLUMNS}
        data={sessions}
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
        renderTopToolbarCustomActions={({ table }) => (
          <Stack direction="row" alignItems="center" gap={2}>
            {table.getIsSomeRowsSelected() && (
              <Button
                startIcon={<StopCircleOutlinedIcon />}
                onClick={openStopSelectedModal}
              >
                {Messages.stopSelected}
              </Button>
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
        onCreateSession={handleCreateSessions}
      />
    </Stack>
  );
};

export default SessionsTable;
