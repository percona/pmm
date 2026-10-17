import {
  Box,
  Button,
  LinearProgress,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Typography,
} from '@mui/material';
import { FC, Fragment, useState } from 'react';
import { LogFilter, LogLine } from 'api/logs';
import { useLogSearch } from 'hooks/api/useLogs';
import { dayLabel } from '../utils/logTime';
import { LogContextDialog } from './LogContextDialog';
import { LogLineRow } from './LogLineRow';
import { LogsErrorAlert } from './LogsErrorAlert';

const COLUMNS = 4;

export const LogsLinesTab: FC<{ filter: LogFilter }> = ({ filter }) => {
  const search = useLogSearch(filter);
  const [contextLine, setContextLine] = useState<LogLine | null>(null);
  const lines = search.data?.pages.flatMap((p) => p.lines) ?? [];

  if (search.error) return <LogsErrorAlert error={search.error} />;
  if (search.isLoading) return <LinearProgress />;
  if (!lines.length) {
    return (
      <Typography color="text.secondary" sx={{ p: 2 }}>
        No log lines match.
      </Typography>
    );
  }

  return (
    <>
      <Paper
        variant="outlined"
        data-testid="logs-lines-tab"
        // clip, not hidden: hidden would make the Paper the sticky header's scroll container.
        sx={{ borderRadius: 3, overflow: 'clip' }}
      >
        <Table
          size="small"
          stickyHeader
          sx={{
            '& th': {
              py: 1,
              bgcolor: 'background.paper',
              borderColor: 'divider',
              color: 'text.secondary',
              fontSize: 11,
              fontWeight: 500,
              letterSpacing: '0.06em',
              textTransform: 'uppercase',
            },
          }}
        >
          <TableHead>
            <TableRow>
              <TableCell>Time</TableCell>
              <TableCell>Level</TableCell>
              <TableCell>Node · Source</TableCell>
              <TableCell>Message</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {lines.map((l, i) => {
              const day = dayLabel(l.time);
              return (
                <Fragment key={`${l.time}-${i}`}>
                  {(i === 0 || day !== dayLabel(lines[i - 1].time)) && (
                    <TableRow data-testid="log-day-separator">
                      <TableCell
                        colSpan={COLUMNS}
                        sx={{
                          py: 0.75,
                          fontSize: 12,
                          color: 'text.secondary',
                          bgcolor: 'background.default',
                          borderColor: 'divider',
                        }}
                      >
                        {day}
                        {i === 0 && ' · local time, hover a time for UTC'}
                      </TableCell>
                    </TableRow>
                  )}
                  <LogLineRow
                    line={l}
                    highlightText={filter.text}
                    onShowContext={setContextLine}
                  />
                </Fragment>
              );
            })}
          </TableBody>
        </Table>
        {search.hasNextPage && (
          <Box sx={{ py: 1.5, textAlign: 'center' }}>
            <Button
              variant="outlined"
              size="small"
              onClick={() => search.fetchNextPage()}
              disabled={search.isFetchingNextPage}
            >
              Load older
            </Button>
          </Box>
        )}
      </Paper>
      <LogContextDialog
        line={contextLine}
        onClose={() => setContextLine(null)}
      />
    </>
  );
};
