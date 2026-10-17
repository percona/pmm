import {
  alpha,
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  LinearProgress,
  useTheme,
} from '@mui/material';
import { FC, useEffect, useRef } from 'react';
import { LogLine, severityClass, severityLabel } from 'api/logs';
import { useLogContext } from 'hooks/api/useLogs';
import { MONO, severityColor } from '../logsTheme';
import { timeOfDay } from '../utils/logTime';
import { LogsErrorAlert } from './LogsErrorAlert';

interface LogContextDialogProps {
  line: LogLine | null;
  onClose: () => void;
}

// Lists cut bodies to 4 KiB and context to 64 KiB, so a cut list body is a prefix of the context one.
const isSameLine = (a: LogLine, b: LogLine) =>
  a.time === b.time &&
  a.severityNumber === b.severityNumber &&
  JSON.stringify(a.attributes) === JSON.stringify(b.attributes) &&
  (b.bodyTruncated ? a.body.startsWith(b.body) : a.body === b.body);

/** The lines before and after one line of the same node and file. */
export const LogContextDialog: FC<LogContextDialogProps> = ({
  line,
  onClose,
}) => {
  const { palette } = useTheme();
  const context = useLogContext(line);
  const anchorRef = useRef<HTMLDivElement>(null);
  // Lines sharing the clicked line's timestamp are all returned; the server can only anchor the first of them.
  const matched = line
    ? (context.data?.lines.findIndex((l) => isSameLine(l, line)) ?? -1)
    : -1;
  const anchorIndex = matched >= 0 ? matched : context.data?.anchorIndex;

  useEffect(() => {
    anchorRef.current?.scrollIntoView?.({ block: 'center' });
  }, [context.data]);

  return (
    <Dialog open={!!line} onClose={onClose} maxWidth="lg" fullWidth>
      <DialogTitle>
        {line?.fileName} on {line?.nodeName}
      </DialogTitle>
      <DialogContent dividers>
        {context.isLoading && <LinearProgress />}
        {context.error && <LogsErrorAlert error={context.error} />}
        {context.data?.lines.map((l, i) => {
          const anchor = i === anchorIndex;
          return (
            <Box
              key={`${l.time}-${i}`}
              ref={anchor ? anchorRef : undefined}
              data-testid={anchor ? 'log-context-anchor' : 'log-context-line'}
              sx={{
                ...MONO,
                display: 'grid',
                gridTemplateColumns: 'auto 48px minmax(0, 1fr)',
                columnGap: 1.5,
                px: 1.5,
                py: 0.25,
                borderLeft: '3px solid',
                borderColor: anchor ? 'primary.main' : 'transparent',
                bgcolor: anchor ? alpha(palette.primary.main, 0.14) : undefined,
              }}
            >
              <Box
                component="span"
                sx={{
                  color: 'text.secondary',
                  fontVariantNumeric: 'tabular-nums',
                }}
              >
                {timeOfDay(l.time)}
              </Box>
              <Box
                component="span"
                sx={{
                  fontSize: 12,
                  fontWeight: 600,
                  color: severityColor(
                    severityClass(l.severityNumber),
                    palette.mode
                  ),
                }}
              >
                {severityLabel(l.severityNumber) || '—'}
              </Box>
              <Box
                component="span"
                sx={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}
              >
                {l.body}
              </Box>
            </Box>
          );
        })}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Close</Button>
      </DialogActions>
    </Dialog>
  );
};
