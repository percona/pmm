import TrackChanges from '@mui/icons-material/TrackChanges';
import Box from '@mui/material/Box';
import List from '@mui/material/List';
import ListItem from '@mui/material/ListItem';
import ListItemIcon from '@mui/material/ListItemIcon';
import ListSubheader from '@mui/material/ListSubheader';
import Paper from '@mui/material/Paper';
import Typography from '@mui/material/Typography';
import { QueryStatsIcon, StatusIcon, Tooltip } from '@percona/peak-ui';
import { type FC, useId } from 'react';
import { Messages } from '../hub.messages';
import type { HubQueries, HubQueryRow, HubSectionState } from '../hub.types';
import { DoorButton } from './DoorLink';
import { HubSection, SectionState } from './HubSection';
import { focusableText } from './hub.sx';

const QueryRow: FC<{ row: HubQueryRow; divider: boolean }> = ({
  row,
  divider,
}) => (
  <ListItem divider={divider} alignItems="flex-start" sx={{ gap: 2 }}>
    <ListItemIcon sx={{ minWidth: 0 }}>
      <StatusIcon
        size="small"
        severity={row.severity}
        titleAccess={Messages.attention.severity[row.severity]}
      />
    </ListItemIcon>
    <Box
      sx={{
        flex: 1,
        minWidth: 0,
        display: 'flex',
        flexWrap: 'wrap',
        alignItems: 'center',
        columnGap: 2,
        rowGap: 0.5,
      }}
    >
      <Box sx={{ flex: '1 1 240px', minWidth: 0 }}>
        <Box sx={{ display: 'flex', alignItems: 'baseline', gap: 1 }}>
          <Tooltip title={row.text} describeChild>
            <Typography
              variant="body2"
              noWrap
              tabIndex={0}
              // Peak's inline code breaks spaces; the row shows one line and the tooltip the rest.
              sx={[
                { minWidth: 0, '& code': { whiteSpace: 'nowrap' } },
                focusableText,
              ]}
            >
              <code>{row.text}</code>
            </Typography>
          </Tooltip>
          <Typography variant="body2" sx={{ flexShrink: 0 }}>
            {row.headline}
          </Typography>
        </Box>
        <Typography variant="body2" color="text.secondary">
          {row.detail}
        </Typography>
      </Box>
      <DoorButton door={row.door} />
    </Box>
  </ListItem>
);

const QueryGroup: FC<{ title: string; rows: HubQueryRow[]; last: boolean }> = ({
  title,
  rows,
  last,
}) => {
  const id = useId();

  return (
    <List
      disablePadding
      aria-labelledby={id}
      subheader={
        <ListSubheader id={id} disableSticky>
          {title}
        </ListSubheader>
      }
    >
      {rows.map((row, index) => (
        <QueryRow
          key={row.id}
          row={row}
          divider={!last || index < rows.length - 1}
        />
      ))}
    </List>
  );
};

export interface QueriesSectionProps {
  queries: HubQueries;
  state: HubSectionState;
  /** Nothing is monitored, so there is no query to vouch for. */
  empty?: boolean;
  onRetry: () => void;
}

/** Queries: the two ways into query data, then only the queries that are off. */
export const QueriesSection: FC<QueriesSectionProps> = ({
  queries,
  state,
  empty = false,
  onRetry,
}) => {
  const { links, slower, runningLong } = queries;

  return (
    <HubSection id="queries" title={Messages.sections.queries}>
      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
        <DoorButton
          door={links.analytics}
          variant="outlined"
          startIcon={<QueryStatsIcon />}
        />
        <DoorButton
          door={links.realtime}
          variant="outlined"
          startIcon={<TrackChanges />}
        />
      </Box>
      <SectionState state={state} onRetry={onRetry} skeletonRows={2}>
        {slower.length || runningLong.length ? (
          <Paper variant="outlined">
            {slower.length > 0 && (
              <QueryGroup
                title={Messages.queries.slower}
                rows={slower}
                last={runningLong.length === 0}
              />
            )}
            {runningLong.length > 0 && (
              <QueryGroup
                title={Messages.queries.runningLong}
                rows={runningLong}
                last
              />
            )}
          </Paper>
        ) : empty ? null : (
          <Typography variant="body2" color="text.secondary">
            {Messages.queries.none}
          </Typography>
        )}
      </SectionState>
    </HubSection>
  );
};
