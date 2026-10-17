import {
  Box,
  Button,
  Chip,
  ChipProps,
  Stack,
  TableCell,
  TableRow,
  Tooltip,
  Typography,
} from '@mui/material';
import { FC, ReactNode, useState } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import { LogLine, severityClass, SeverityClass } from 'api/logs';
import {
  assistantAskUrl,
  DATABASE_PRESETS,
  nodeDashboardUrl,
  qanUrl,
} from '../utils/logLinks';

const CHIP_COLORS: Record<SeverityClass, ChipProps['color']> = {
  error: 'error',
  warning: 'warning',
  info: 'info',
  other: 'default',
};

const LINK_WINDOW_MS = 15 * 60_000;
const COLUMNS = 5;

/** Wraps case-insensitive matches of term in <mark>. */
const highlight = (text: string, term?: string): ReactNode => {
  if (!term) return text;
  const lower = text.toLowerCase();
  const t = term.toLowerCase();
  const parts: ReactNode[] = [];
  let from = 0;
  for (let at = lower.indexOf(t); at >= 0; at = lower.indexOf(t, from)) {
    parts.push(
      text.slice(from, at),
      <mark key={at}>{text.slice(at, at + t.length)}</mark>
    );
    from = at + t.length;
  }
  parts.push(text.slice(from));
  return parts;
};

interface LogLineRowProps {
  line: LogLine;
  highlightText?: string;
  onShowContext: (line: LogLine) => void;
}

export const LogLineRow: FC<LogLineRowProps> = ({
  line,
  highlightText,
  onShowContext,
}) => {
  const [open, setOpen] = useState(false);
  const ms = Date.parse(line.time);
  const [from, to] = [ms - LINK_WINDOW_MS, ms + LINK_WINDOW_MS];
  const firstLine = line.body.split('\n', 1)[0];
  const cls = severityClass(line.severityNumber);

  return (
    <>
      <TableRow
        hover
        onClick={() => setOpen(!open)}
        sx={{ cursor: 'pointer', '& td': { verticalAlign: 'top' } }}
        data-testid="log-line-row"
      >
        <TableCell sx={{ whiteSpace: 'nowrap' }}>
          <Tooltip title={`${line.time} UTC`}>
            <span>{new Date(ms).toLocaleString()}</span>
          </Tooltip>
        </TableCell>
        <TableCell>{line.nodeName}</TableCell>
        <TableCell>{line.source}</TableCell>
        <TableCell>
          <Chip
            size="small"
            label={line.severityText || cls}
            color={CHIP_COLORS[cls]}
          />
        </TableCell>
        <TableCell
          sx={{
            fontFamily: 'monospace',
            wordBreak: 'break-word',
            width: '100%',
          }}
        >
          {highlight(firstLine, highlightText)}
        </TableCell>
      </TableRow>
      {open && (
        <TableRow>
          <TableCell colSpan={COLUMNS} data-testid="log-line-details">
            <Stack spacing={1.5}>
              <Box
                component="pre"
                sx={{
                  m: 0,
                  fontFamily: 'monospace',
                  whiteSpace: 'pre-wrap',
                  wordBreak: 'break-word',
                }}
              >
                {highlight(line.body, highlightText)}
              </Box>
              {line.bodyTruncated && (
                <Typography variant="caption" color="text.secondary">
                  truncated — open context for the full line
                </Typography>
              )}
              <Stack direction="row" sx={{ flexWrap: 'wrap', gap: 0.5 }}>
                {Object.entries(line.attributes).map(([k, v]) => (
                  <Chip
                    key={k}
                    size="small"
                    variant="outlined"
                    label={`${k}: ${v}`}
                  />
                ))}
              </Stack>
              <Stack direction="row" spacing={1}>
                <Button size="small" onClick={() => onShowContext(line)}>
                  Show context
                </Button>
                <Button
                  size="small"
                  component={RouterLink}
                  to={assistantAskUrl(line, from, to)}
                >
                  Ask Assistant
                </Button>
                <Button
                  size="small"
                  component={RouterLink}
                  to={nodeDashboardUrl(line.nodeName, from, to)}
                >
                  Open dashboard
                </Button>
                {DATABASE_PRESETS.includes(line.preset) && (
                  <Button size="small" href={qanUrl(line.nodeName, from, to)}>
                    Open QAN
                  </Button>
                )}
              </Stack>
            </Stack>
          </TableCell>
        </TableRow>
      )}
    </>
  );
};
