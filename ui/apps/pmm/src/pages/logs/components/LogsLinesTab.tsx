import {
  Box,
  Button,
  LinearProgress,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Typography,
} from '@mui/material';
import { FC, useState } from 'react';
import { LogFilter, LogLine } from 'api/logs';
import { useLogSearch } from 'hooks/api/useLogs';
import { LogContextDialog } from './LogContextDialog';
import { LogLineRow } from './LogLineRow';
import { LogsErrorAlert } from './LogsErrorAlert';

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
    <Box data-testid="logs-lines-tab">
      <Table size="small" stickyHeader>
        <TableHead>
          <TableRow>
            <TableCell>Time</TableCell>
            <TableCell>Node</TableCell>
            <TableCell>Source</TableCell>
            <TableCell>Severity</TableCell>
            <TableCell>Message</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {lines.map((l, i) => (
            <LogLineRow
              key={`${l.time}-${i}`}
              line={l}
              highlightText={filter.text}
              onShowContext={setContextLine}
            />
          ))}
        </TableBody>
      </Table>
      {search.hasNextPage && (
        <Box sx={{ py: 1, textAlign: 'center' }}>
          <Button
            onClick={() => search.fetchNextPage()}
            disabled={search.isFetchingNextPage}
          >
            Load older
          </Button>
        </Box>
      )}
      <LogContextDialog
        line={contextLine}
        onClose={() => setContextLine(null)}
      />
    </Box>
  );
};
