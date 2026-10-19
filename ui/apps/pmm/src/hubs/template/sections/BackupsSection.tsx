import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Paper from '@mui/material/Paper';
import Typography from '@mui/material/Typography';
import { useTheme } from '@mui/material/styles';
import { EmptyState, StatusIcon, Table } from '@percona/peak-ui';
import { format, isValid, parseISO } from 'date-fns';
import type { MRT_ColumnDef } from 'material-react-table';
import type { FC, ReactNode } from 'react';
import { Messages } from '../hub.messages';
import type {
  HubBackupRow,
  HubBackups,
  HubMessages,
  HubSectionState,
} from '../hub.types';
import { DoorButton, DoorLink } from './DoorLink';
import { HubSection, SectionState } from './HubSection';
import { visuallyHidden } from './hub.sx';
import { formatAge } from './hub.time';

const NotApplicable: FC = () => (
  <Box component="span" sx={{ color: 'text.secondary' }}>
    <span aria-hidden="true">—</span>
    <Box component="span" sx={visuallyHidden}>
      {Messages.backups.notApplicable}
    </Box>
  </Box>
);

const Flagged: FC<{ children: ReactNode }> = ({ children }) => (
  <Box
    component="span"
    sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.5 }}
  >
    <StatusIcon size="small" severity="warning" />
    {children}
  </Box>
);

const timeOfDay = (iso: string) => {
  const date = parseISO(iso);
  return isValid(date) ? format(date, 'HH:mm') : iso;
};

const LastGoodCell: FC<{ row: HubBackupRow }> = ({ row }) => {
  const { lastGood, failedAt, doors } = row;

  if (failedAt) {
    return (
      // A flex column, so the marker never sits on a text line that its icon would make taller.
      <Box
        sx={{
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'flex-start',
        }}
      >
        <Flagged>
          <DoorLink door={doors.lastGood} color="inherit">
            {Messages.backups.failed(formatAge(failedAt))}
          </DoorLink>
        </Flagged>
        <Typography variant="caption" color="text.secondary" component="div">
          {lastGood
            ? Messages.backups.lastGoodSince(formatAge(lastGood.at))
            : Messages.backups.noneYet}
        </Typography>
      </Box>
    );
  }

  if (!lastGood) {
    return <>{Messages.backups.noneYet}</>;
  }

  const age = (
    <DoorLink door={doors.lastGood} color="inherit">
      {formatAge(lastGood.at)}
      {lastGood.level !== 'ok' && (
        <Box component="span" sx={visuallyHidden}>
          {` ${Messages.measure.over.warning}`}
        </Box>
      )}
    </DoorLink>
  );

  return lastGood.level === 'ok' ? age : <Flagged>{age}</Flagged>;
};

const PitrCell: FC<{ row: HubBackupRow }> = ({ row }) => {
  if (row.pitr === 'notApplicable') {
    return <NotApplicable />;
  }

  const link = (
    <DoorLink door={row.doors.schedule} color="inherit">
      {Messages.backups.pitr[row.pitr]}
    </DoorLink>
  );

  return row.pitr === 'stopped' ? <Flagged>{link}</Flagged> : link;
};

const COLUMNS: MRT_ColumnDef<HubBackupRow>[] = [
  {
    id: 'cluster',
    header: Messages.backups.columns.cluster,
    accessorKey: 'cluster',
    size: 100,
    Cell: ({ row }) => (
      <DoorLink door={row.original.doors.cluster} color="inherit">
        {row.original.cluster}
      </DoorLink>
    ),
  },
  {
    id: 'lastGood',
    header: Messages.backups.columns.lastGood,
    accessorFn: (row) => row.lastGood?.at ?? null,
    size: 160,
    Cell: ({ row }) => <LastGoodCell row={row.original} />,
  },
  {
    id: 'pitr',
    header: Messages.backups.columns.pitr,
    accessorKey: 'pitr',
    size: 140,
    Cell: ({ row }) => <PitrCell row={row.original} />,
  },
  {
    id: 'nextRun',
    header: Messages.backups.columns.nextRun,
    accessorFn: (row) => row.nextRun ?? null,
    size: 100,
    Cell: ({ row }) =>
      row.original.nextRun ? (
        <DoorLink door={row.original.doors.schedule} color="inherit">
          <time dateTime={row.original.nextRun}>
            {timeOfDay(row.original.nextRun)}
          </time>
        </DoorLink>
      ) : (
        <NotApplicable />
      ),
  },
];

export interface BackupsSectionProps {
  technologyId: string;
  backups: HubBackups;
  state: HubSectionState;
  messages: HubMessages;
  /** The environment filter matches no service, so there is nothing to back up here. */
  matchesNothing?: boolean;
  onRetry: () => void;
}

/** Backups: the last good backup per cluster, or how to start. */
export const BackupsSection: FC<BackupsSectionProps> = ({
  technologyId,
  backups,
  state,
  messages,
  matchesNothing = false,
  onRetry,
}) => {
  const theme = useTheme();
  const { doors } = backups;
  const setUp = doors.setUp && (
    <DoorButton door={doors.setUp} variant="outlined" />
  );

  const body = (() => {
    switch (backups.state) {
      // The section's warning says why there is nothing to show.
      case 'unknown':
        return null;
      case 'featureOff':
        return (
          <Alert
            severity="info"
            action={
              doors.settings && (
                <DoorButton door={doors.settings} color="inherit" />
              )
            }
          >
            {Messages.backups.featureOff}
          </Alert>
        );
      case 'noServices':
        return matchesNothing ? (
          <Typography variant="body2" color="text.secondary">
            {Messages.environment.noService(messages.name)}
          </Typography>
        ) : (
          <Paper variant="outlined">
            <EmptyState
              size="small"
              titleComponent="h3"
              title={messages.backups.emptyTitle}
              description={messages.backups.noServices}
            />
          </Paper>
        );
      case 'notSetUp':
        return (
          <Paper variant="outlined">
            <EmptyState
              size="small"
              titleComponent="h3"
              title={messages.backups.emptyTitle}
              description={
                backups.hint
                  ? `${messages.backups.emptyDescription} ${backups.hint}.`
                  : messages.backups.emptyDescription
              }
              actions={setUp}
            />
          </Paper>
        );
      default:
        return (
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
            <Table<HubBackupRow>
              tableName={`hub-${technologyId}-backups`}
              data={backups.rows}
              columns={COLUMNS}
              getRowId={(row) => row.id}
              enablePagination={false}
              enableSorting={false}
              enableColumnActions={false}
              enableColumnFilters={false}
              enableGlobalFilter={false}
              enableHiding={false}
              enableTopToolbar={false}
              enableBottomToolbar={false}
              initialState={{ density: 'compact' }}
              muiTableProps={{ 'aria-label': Messages.backups.tableLabel }}
              muiTablePaperProps={{
                variant: 'outlined',
                sx: {
                  borderRadius: `${theme.shape.borderRadiusMd}px`,
                  overflow: 'hidden',
                },
              }}
              muiTableContainerProps={{ sx: { borderBottom: 'none' } }}
            />
            {backups.hint && (
              <Typography variant="body2" color="text.secondary">
                {backups.hint}
              </Typography>
            )}
            <Box
              sx={{
                display: 'flex',
                flexWrap: 'wrap',
                alignItems: 'center',
                gap: 1,
              }}
            >
              <DoorButton door={doors.dashboard} />
              {setUp}
            </Box>
          </Box>
        );
    }
  })();

  return (
    <HubSection id="backups" title={Messages.sections.backups} door={doors.app}>
      <SectionState state={state} onRetry={onRetry}>
        {body}
      </SectionState>
    </HubSection>
  );
};
