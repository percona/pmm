import { Box, Button, Chip, Stack, Typography } from '@mui/material';
import { FC } from 'react';
import { LogFilter, LogLine, LogPattern } from 'api/logs';
import { useLogPatternDetails } from 'hooks/api/useLogs';
import { MONO } from '../logsTheme';

const formatWhen = (iso: string) =>
  new Date(iso).toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  });

/** A pattern's count against the previous range of the same length, in words. */
const changeText = ({
  count,
  previousCount,
}: Pick<LogPattern, 'count' | 'previousCount'>): string => {
  if (!previousCount) return 'none in the previous range';
  const ratio = count / previousCount;
  if (ratio >= 1.1) {
    const times =
      ratio < 10 ? ratio.toFixed(1).replace(/\.0$/, '') : Math.round(ratio);
    return `${times}× the previous range`;
  }
  if (ratio <= 0.9) {
    return `${Math.round((1 - ratio) * 100)}% fewer than the previous range`;
  }
  return 'about the same as the previous range';
};

interface LogPatternSummaryProps {
  line: LogLine;
  filter: LogFilter;
  /** Applies Show only and Hide; the buttons show only when set. */
  onFilterChange?: (next: LogFilter) => void;
}

/** How often a line's pattern was logged in the time range, across all nodes and sources. */
export const LogPatternSummary: FC<LogPatternSummaryProps> = ({
  line,
  filter,
  onFilterChange,
}) => {
  const details = useLogPatternDetails(
    filter.start,
    filter.end,
    line.patternHash || undefined
  );
  if (!line.patternHash) return null;
  const p = details.data;

  return (
    <Box
      data-testid="log-pattern-summary"
      sx={{
        p: 1.5,
        border: 1,
        borderColor: 'divider',
        borderRadius: 2,
        bgcolor: 'background.default',
      }}
    >
      <Stack direction="row" alignItems="center" spacing={1} sx={{ mb: 0.5 }}>
        <Typography
          variant="caption"
          color="text.secondary"
          sx={{ textTransform: 'uppercase', letterSpacing: '0.05em' }}
        >
          Pattern
        </Typography>
        {p?.isNew && (
          <Chip size="small" color="warning" variant="outlined" label="NEW" />
        )}
        {onFilterChange && (
          <Stack direction="row" spacing={1} sx={{ ml: 'auto !important' }}>
            <Button
              size="small"
              disabled={!p}
              onClick={() =>
                p && onFilterChange({ ...filter, pattern: p.pattern })
              }
            >
              Show only this pattern
            </Button>
            <Button
              size="small"
              onClick={() =>
                onFilterChange({
                  ...filter,
                  excludePatternHashes: [
                    ...(filter.excludePatternHashes ?? []),
                    line.patternHash,
                  ],
                })
              }
            >
              Hide this pattern
            </Button>
          </Stack>
        )}
      </Stack>
      {details.isLoading && (
        <Typography variant="body2" color="text.secondary">
          Counting this pattern…
        </Typography>
      )}
      {details.error && (
        <Typography variant="body2" color="error">
          Could not count this pattern.
        </Typography>
      )}
      {p && (
        <>
          <Typography sx={{ ...MONO, mb: 0.75, wordBreak: 'break-word' }}>
            {p.pattern}
          </Typography>
          <Typography variant="body2" color="text.secondary">
            <Box
              component="span"
              sx={{ color: 'text.primary', fontWeight: 600 }}
            >
              {p.count.toLocaleString()} lines
            </Box>{' '}
            in this range across all nodes · {changeText(p)} · first{' '}
            {formatWhen(p.firstSeen)} · last {formatWhen(p.lastSeen)}
            {p.topSources.length > 0 && (
              <> · most from {p.topSources.join(', ')}</>
            )}
          </Typography>
        </>
      )}
    </Box>
  );
};
