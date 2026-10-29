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
import { FC, Fragment, useEffect, useRef, useState } from 'react';
import { LogFilter, LogLine } from 'api/logs';
import { useLogSearch } from 'hooks/api/useLogs';
import { lineKey } from '../utils/logDisplay';
import { dayLabel } from '../utils/logTime';
import { LogContextDialog } from './LogContextDialog';
import { LogLineRow } from './LogLineRow';
import { LogsErrorAlert } from './LogsErrorAlert';

const COLUMNS = 4;
// Older lines load when the end of the list comes this close.
const LOAD_AHEAD = '600px';

interface LogsLinesTabProps {
  filter: LogFilter;
  /** lineKey of the line the chart follows. */
  focusedKey?: string;
  onFocus?: (line: LogLine | null) => void;
  onFilterChange?: (next: LogFilter) => void;
}

export const LogsLinesTab: FC<LogsLinesTabProps> = ({
  filter,
  focusedKey,
  onFocus,
  onFilterChange,
}) => {
  const search = useLogSearch(filter);
  const [contextLine, setContextLine] = useState<LogLine | null>(null);
  const lines = search.data?.pages.flatMap((p) => p.lines) ?? [];
  const scroller = useRef<HTMLDivElement>(null);
  const end = useRef<HTMLDivElement>(null);
  const { hasNextPage, isFetchingNextPage, fetchNextPage } = search;

  useEffect(() => {
    if (!end.current || typeof IntersectionObserver === 'undefined') return;
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting && hasNextPage && !isFetchingNextPage) {
          fetchNextPage();
        }
      },
      { root: scroller.current, rootMargin: LOAD_AHEAD }
    );
    observer.observe(end.current);
    return () => observer.disconnect();
  }, [hasNextPage, isFetchingNextPage, fetchNextPage, lines.length]);

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
        ref={scroller}
        variant="outlined"
        data-testid="logs-lines-tab"
        // The list scrolls on its own, under the pinned filters and chart; the header sticks to its top.
        sx={{ borderRadius: 3, overflow: 'auto', flex: 1, minHeight: 0 }}
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
              <TableCell>Node · Service</TableCell>
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
                    filter={filter}
                    highlightText={filter.text}
                    onShowContext={setContextLine}
                    focused={!!focusedKey && focusedKey === lineKey(l)}
                    onFocus={onFocus}
                    onFilterChange={onFilterChange}
                  />
                </Fragment>
              );
            })}
          </TableBody>
        </Table>
        <div ref={end} />
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
