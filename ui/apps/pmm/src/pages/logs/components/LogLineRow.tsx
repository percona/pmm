import {
  alpha,
  Box,
  Button,
  Stack,
  TableCell,
  TableRow,
  Theme,
  Tooltip,
  Typography,
  useTheme,
} from '@mui/material';
import { FC, ReactNode, useState } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import { LogLine, severityClass, severityLabel } from 'api/logs';
import { MONO, severityColor } from '../logsTheme';
import { useManagedServices } from 'hooks/api/useServices';
import {
  assistantAskUrl,
  dashboardUrl,
  DATABASE_PRESETS,
  qanUrl,
  serviceTypeOf,
} from '../utils/logLinks';
import { detailJson, previewText } from '../utils/logDisplay';
import { timeOfDay } from '../utils/logTime';

const LINK_WINDOW_MS = 15 * 60_000;
const COLUMNS = 4;

const markSx = (theme: Theme) => {
  const warning = severityColor('warning', theme.palette.mode);
  return {
    bgcolor: alpha(warning, 0.22),
    color: theme.palette.mode === 'dark' ? '#fde68a' : 'inherit',
    borderRadius: '3px',
    boxShadow: `inset 0 0 0 1px ${alpha(warning, 0.4)}`,
  };
};

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
      <Box component="mark" key={at} sx={markSx}>
        {text.slice(at, at + t.length)}
      </Box>
    );
    from = at + t.length;
  }
  parts.push(text.slice(from));
  return parts;
};

const ellipsis = {
  overflow: 'hidden',
  textOverflow: 'ellipsis',
  whiteSpace: 'nowrap',
} as const;

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
  const { palette } = useTheme();
  const [open, setOpen] = useState(false);
  const ms = Date.parse(line.time);
  const [from, to] = [ms - LINK_WINDOW_MS, ms + LINK_WINDOW_MS];
  const color = severityColor(severityClass(line.severityNumber), palette.mode);
  const label = severityLabel(line.severityNumber);
  const serviceType = serviceTypeOf(line.preset);
  const services = useManagedServices(
    {},
    { enabled: open && !!serviceType, staleTime: 60_000 }
  );
  const service = services.data?.services.find(
    (s) => s.nodeName === line.nodeName && s.serviceType === serviceType
  );
  const json = detailJson(line);
  const attributes = [
    ...(line.severityText && line.severityText !== label
      ? [['original level', line.severityText]]
      : []),
    // attr is shown as JSON below the message; MongoDB's s repeats the original level.
    ...Object.entries(line.attributes).filter(
      ([k, v]) =>
        v !== '' &&
        !(k === 'attr' && json) &&
        !(k === 's' && v === line.severityText)
    ),
  ];

  return (
    <>
      <TableRow
        hover
        onClick={() => setOpen(!open)}
        sx={{
          cursor: 'pointer',
          bgcolor: open ? 'action.hover' : undefined,
          '& td': { py: 0.75, borderColor: 'divider' },
        }}
        data-testid="log-line-row"
      >
        <TableCell
          sx={{
            ...MONO,
            whiteSpace: 'nowrap',
            color: 'text.secondary',
            fontVariantNumeric: 'tabular-nums',
            boxShadow: `inset 3px 0 0 ${color}`,
          }}
        >
          <Tooltip
            title={
              <>
                <div>{new Date(ms).toLocaleString()}</div>
                <div>{line.time} UTC</div>
              </>
            }
          >
            <span>{timeOfDay(line.time)}</span>
          </Tooltip>
        </TableCell>
        <TableCell
          sx={{ ...MONO, fontSize: 12, fontWeight: 600, color, ...ellipsis }}
        >
          {label || '—'}
        </TableCell>
        <TableCell
          data-testid="log-line-origin"
          sx={{ maxWidth: 260, ...ellipsis }}
        >
          {line.nodeName}
          {line.source && line.source !== line.nodeName && (
            <Box component="span" sx={{ color: 'text.secondary' }}>
              {' · '}
              {line.source}
            </Box>
          )}
        </TableCell>
        <TableCell sx={{ ...MONO, maxWidth: 0, width: '100%', ...ellipsis }}>
          {highlight(previewText(line), highlightText)}
        </TableCell>
      </TableRow>
      {open && (
        <TableRow sx={{ bgcolor: 'action.hover' }}>
          <TableCell
            colSpan={COLUMNS}
            data-testid="log-line-details"
            sx={{ borderColor: 'divider', px: 2, pt: 0.5, pb: 2 }}
          >
            <Stack spacing={2}>
              <Box
                component="pre"
                sx={{
                  ...MONO,
                  m: 0,
                  p: 1.5,
                  lineHeight: 1.6,
                  whiteSpace: 'pre-wrap',
                  wordBreak: 'break-word',
                  maxHeight: 480,
                  overflow: 'auto',
                  bgcolor: 'background.default',
                  border: 1,
                  borderColor: 'divider',
                  borderRadius: 2,
                }}
              >
                {highlight(line.body, highlightText)}
              </Box>
              {json && (
                <Box
                  component="pre"
                  data-testid="log-line-attr"
                  sx={{
                    ...MONO,
                    m: 0,
                    p: 1.5,
                    lineHeight: 1.5,
                    whiteSpace: 'pre-wrap',
                    wordBreak: 'break-word',
                    maxHeight: 360,
                    overflow: 'auto',
                    color: 'text.secondary',
                    bgcolor: 'background.default',
                    border: 1,
                    borderColor: 'divider',
                    borderRadius: 2,
                  }}
                >
                  {json}
                </Box>
              )}
              {line.bodyTruncated && (
                <Typography variant="caption" color="text.secondary">
                  truncated — open context for the full line
                </Typography>
              )}
              {attributes.length > 0 && (
                <Box
                  sx={{
                    display: 'grid',
                    gridTemplateColumns:
                      'repeat(auto-fill, minmax(200px, 1fr))',
                    gap: 1.5,
                  }}
                >
                  {attributes.map(([k, v]) => (
                    <Box key={k} sx={{ minWidth: 0 }}>
                      <Typography
                        variant="caption"
                        color="text.secondary"
                        sx={{
                          display: 'block',
                          textTransform: 'uppercase',
                          letterSpacing: '0.05em',
                        }}
                      >
                        {k}
                      </Typography>
                      <Typography title={v} sx={{ ...MONO, ...ellipsis }}>
                        {v}
                      </Typography>
                    </Box>
                  ))}
                </Box>
              )}
              <Stack direction="row" spacing={1}>
                <Button
                  size="small"
                  variant="outlined"
                  onClick={() => onShowContext(line)}
                >
                  Show context
                </Button>
                <Button
                  size="small"
                  variant="contained"
                  disableElevation
                  component={RouterLink}
                  to={assistantAskUrl(line, from, to)}
                >
                  Ask Assistant
                </Button>
                <Button
                  size="small"
                  variant="outlined"
                  component={RouterLink}
                  to={dashboardUrl(line, from, to, service?.serviceName)}
                  target="_blank"
                  rel="noopener"
                >
                  Open dashboard
                </Button>
                {DATABASE_PRESETS.includes(line.preset) && (
                  <Button
                    size="small"
                    variant="outlined"
                    component={RouterLink}
                    to={qanUrl(line, from, to)}
                    target="_blank"
                    rel="noopener"
                  >
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
