import {
  Box,
  Chip,
  ChipProps,
  LinearProgress,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Typography,
} from '@mui/material';
import { FC, Fragment } from 'react';
import { LogFilter, severityClass, SeverityClass } from 'api/logs';
import { useLogPatterns } from 'hooks/api/useLogs';
import { LogsErrorAlert } from './LogsErrorAlert';

const CHIP_COLORS: Record<SeverityClass, ChipProps['color']> = {
  error: 'error',
  warning: 'warning',
  info: 'info',
  other: 'default',
};

/** A pattern with its `?` placeholders dimmed. */
const PatternText: FC<{ pattern: string }> = ({ pattern }) => (
  <>
    {pattern.split('?').map((part, i) => (
      <Fragment key={i}>
        {i > 0 && (
          <Box component="span" sx={{ color: 'text.disabled' }}>
            ?
          </Box>
        )}
        {part}
      </Fragment>
    ))}
  </>
);

interface LogsPatternsTabProps {
  filter: LogFilter;
  onSelect: (pattern: string) => void;
}

export const LogsPatternsTab: FC<LogsPatternsTabProps> = ({
  filter,
  onSelect,
}) => {
  const patterns = useLogPatterns(filter, true);

  if (patterns.error) return <LogsErrorAlert error={patterns.error} />;
  if (patterns.isLoading) return <LinearProgress />;
  if (!patterns.data?.length) {
    return (
      <Typography color="text.secondary" sx={{ p: 2 }}>
        No log lines match.
      </Typography>
    );
  }

  return (
    <Table size="small" stickyHeader data-testid="logs-patterns-tab">
      <TableHead>
        <TableRow>
          <TableCell>Pattern</TableCell>
          <TableCell align="right">Count</TableCell>
          <TableCell>Max severity</TableCell>
          <TableCell>First seen</TableCell>
          <TableCell>Last seen</TableCell>
        </TableRow>
      </TableHead>
      <TableBody>
        {patterns.data.map((p) => {
          const cls = severityClass(p.maxSeverityNumber);
          return (
            <TableRow
              key={p.pattern}
              hover
              onClick={() => onSelect(p.pattern)}
              sx={{ cursor: 'pointer', '& td': { verticalAlign: 'top' } }}
              data-testid="log-pattern-row"
            >
              <TableCell
                sx={{ fontFamily: 'monospace', wordBreak: 'break-word' }}
              >
                {p.isNew && (
                  <Chip
                    size="small"
                    color="warning"
                    variant="outlined"
                    label="new"
                    sx={{ mr: 1 }}
                  />
                )}
                <PatternText pattern={p.pattern} />
              </TableCell>
              <TableCell align="right">{p.count.toLocaleString()}</TableCell>
              <TableCell>
                <Chip size="small" label={cls} color={CHIP_COLORS[cls]} />
              </TableCell>
              <TableCell sx={{ whiteSpace: 'nowrap' }}>
                {new Date(p.firstSeen).toLocaleString()}
              </TableCell>
              <TableCell sx={{ whiteSpace: 'nowrap' }}>
                {new Date(p.lastSeen).toLocaleString()}
              </TableCell>
            </TableRow>
          );
        })}
      </TableBody>
    </Table>
  );
};
