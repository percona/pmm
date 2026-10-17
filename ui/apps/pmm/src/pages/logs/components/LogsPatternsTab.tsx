import {
  alpha,
  Box,
  LinearProgress,
  Paper,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Typography,
  useTheme,
} from '@mui/material';
import { FC, Fragment } from 'react';
import { LogFilter, severityClass, severityLabel } from 'api/logs';
import { useLogPatterns } from 'hooks/api/useLogs';
import { MONO, severityColor } from '../logsTheme';
import { shortDateTime } from '../utils/logTime';
import { LogsErrorAlert } from './LogsErrorAlert';

/** A pattern with its `?` placeholders shown as tokens. */
const PatternText: FC<{ pattern: string }> = ({ pattern }) => (
  <>
    {pattern.split('?').map((part, i) => (
      <Fragment key={i}>
        {i > 0 && (
          <Box
            component="span"
            data-testid="log-pattern-placeholder"
            sx={(theme) => ({
              mx: 0.25,
              px: 0.5,
              borderRadius: '4px',
              fontWeight: 600,
              color: 'primary.main',
              bgcolor: alpha(theme.palette.primary.main, 0.12),
              boxShadow: `inset 0 0 0 1px ${alpha(theme.palette.primary.main, 0.3)}`,
            })}
          >
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
  const { palette } = useTheme();
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

  const top = Math.max(...patterns.data.map((p) => p.count));

  return (
    <Paper variant="outlined" sx={{ borderRadius: 3, overflow: 'clip' }}>
      <Table
        size="small"
        stickyHeader
        data-testid="logs-patterns-tab"
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
            <TableCell>Lines</TableCell>
            <TableCell>Level</TableCell>
            <TableCell>Pattern</TableCell>
            <TableCell>First seen</TableCell>
            <TableCell>Last seen</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {patterns.data.map((p) => {
            const color = severityColor(
              severityClass(p.maxSeverityNumber),
              palette.mode
            );
            const share = Math.round((p.count / top) * 100);
            return (
              <TableRow
                key={p.pattern}
                hover
                onClick={() => onSelect(p.pattern)}
                sx={{ cursor: 'pointer', '& td': { borderColor: 'divider' } }}
                data-testid="log-pattern-row"
              >
                <TableCell
                  sx={{ width: 140, boxShadow: `inset 3px 0 0 ${color}` }}
                >
                  <Stack spacing={0.5}>
                    <Typography
                      variant="body2"
                      sx={{
                        fontWeight: 600,
                        fontVariantNumeric: 'tabular-nums',
                      }}
                    >
                      {p.count.toLocaleString()}
                    </Typography>
                    <Box
                      role="meter"
                      aria-label="Lines compared with the most frequent pattern"
                      aria-valuenow={share}
                      aria-valuemin={0}
                      aria-valuemax={100}
                      sx={{
                        height: 3,
                        borderRadius: 2,
                        bgcolor: 'action.hover',
                        overflow: 'hidden',
                      }}
                    >
                      <Box
                        sx={{
                          width: `${share}%`,
                          height: '100%',
                          bgcolor: 'text.secondary',
                        }}
                      />
                    </Box>
                  </Stack>
                </TableCell>
                <TableCell
                  sx={{ ...MONO, fontSize: 12, fontWeight: 600, color }}
                >
                  {severityLabel(p.maxSeverityNumber) || '—'}
                </TableCell>
                <TableCell sx={{ ...MONO, wordBreak: 'break-word' }}>
                  {p.isNew && (
                    <Box
                      component="span"
                      sx={{
                        mr: 1,
                        px: 0.5,
                        fontSize: 11,
                        fontWeight: 700,
                        letterSpacing: '0.06em',
                        textTransform: 'uppercase',
                        borderRadius: '4px',
                        color: severityColor('warning', palette.mode),
                        boxShadow: `inset 0 0 0 1px ${alpha(severityColor('warning', palette.mode), 0.5)}`,
                      }}
                    >
                      new
                    </Box>
                  )}
                  <PatternText pattern={p.pattern} />
                </TableCell>
                <TableCell
                  sx={{
                    whiteSpace: 'nowrap',
                    color: 'text.secondary',
                    fontVariantNumeric: 'tabular-nums',
                  }}
                >
                  {shortDateTime(p.firstSeen)}
                </TableCell>
                <TableCell
                  sx={{
                    whiteSpace: 'nowrap',
                    color: 'text.secondary',
                    fontVariantNumeric: 'tabular-nums',
                  }}
                >
                  {shortDateTime(p.lastSeen)}
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </Paper>
  );
};
