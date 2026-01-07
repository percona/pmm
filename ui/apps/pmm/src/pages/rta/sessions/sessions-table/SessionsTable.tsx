import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import { FC, useState } from 'react';
import { Messages } from './SessionsTable.messages';
import StopCircleOutlinedIcon from '@mui/icons-material/StopCircleOutlined';
import AddOutlinedIcon from '@mui/icons-material/AddOutlined';
import { Table } from '@percona/ui-lib';
import {
  boxClasses,
  paperClasses,
  Skeleton,
  toolbarClasses,
} from '@mui/material';
import { SESSIONS_TABLE_COLUMNS } from './SessionsTable.constants';
import { useRealTimeAgents } from 'hooks/api/useRealTime';
import { getSessions } from './SessionsTable.utils';
import { RealTimeSession } from 'types/rta.types';

const SessionsTable: FC = () => {
  const { data: agents = [], isLoading } = useRealTimeAgents({
    refetchInterval: 5000,
  });
  const sessions = getSessions(agents);
  const [openDetailPanels, setOpenDetailPanels] = useState<
    Record<string, boolean>
  >({});

  const handleStop = (session: RealTimeSession) => {
    setOpenDetailPanels((prev) => ({
      ...prev,
      [session.sessionId]: !prev[session.sessionId],
    }));
  };

  const handleStopAll = () => {};

  if (isLoading) {
    return <Skeleton variant="rounded" height="100%" />;
  }

  return (
    <Stack
      sx={{
        flex: 1,
        minHeight: 0,
        overflow: 'hidden',
        [`& > .${paperClasses.root}`]: {
          flex: 1,
          display: 'flex',
          flexDirection: 'column',
          minHeight: 0,
          overflow: 'hidden',
        },
        [`& > .${paperClasses.root} > ${toolbarClasses.root}`]: {
          backgroundColor: 'transparent',
          flexShrink: 0,
        },
        [`& > .${paperClasses.root} > .MuiTableContainer-root`]: {
          flex: 1,
          flexShrink: 1,
          minHeight: 0,
          overflow: 'auto',
        },
        [`& > .${paperClasses.root} > .MuiTablePagination-root,
          & > .${paperClasses.root} > .MuiToolbar-root:has(.MuiTablePagination-root)`]:
          {
            flexShrink: 0,
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
        getRowId={(row) => row.sessionId}
        // todo
        noDataMessage={''}
        tableName="rta-sessions"
        columns={SESSIONS_TABLE_COLUMNS}
        data={sessions}
        enableHiding={false}
        enableGlobalFilter={false}
        enableRowSelection
        enableStickyHeader
        enableSubRowSelection
        enableExpanding
        enableExpandAll
        enableRowActions
        renderRowActions={({ row }) => (
          <Button
            color="inherit"
            size="small"
            onClick={() => handleStop(row.original)}
          >
            {Messages.stop}
          </Button>
        )}
        getSubRows={(row) => row.serviceSessions}
        muiTableContainerProps={{
          sx: {
            overflow: 'auto',
          },
        }}
        muiTopToolbarProps={{
          sx: {
            [`& > .${boxClasses.root}`]: {
              backgroundColor: 'transparent',
              alignItems: 'center',
              flexDirection: 'row-reverse',
            },
          },
        }}
        renderTopToolbarCustomActions={() => (
          <Stack direction="row" alignItems="center" gap={2}>
            <Button startIcon={<StopCircleOutlinedIcon />}>
              {Messages.stopAll}
            </Button>
            <Button startIcon={<AddOutlinedIcon />}>
              {Messages.newSession}
            </Button>
          </Stack>
        )}
      />
    </Stack>
  );
};

export default SessionsTable;
