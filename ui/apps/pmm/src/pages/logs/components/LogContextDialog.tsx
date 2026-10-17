import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  LinearProgress,
} from '@mui/material';
import { FC, useEffect, useRef } from 'react';
import { LogLine } from 'api/logs';
import { useLogContext } from 'hooks/api/useLogs';
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
                fontFamily: 'monospace',
                fontSize: 'body2.fontSize',
                whiteSpace: 'pre-wrap',
                wordBreak: 'break-word',
                px: 1,
                bgcolor: anchor ? 'action.selected' : undefined,
              }}
            >
              {new Date(l.time).toLocaleString()} {l.severityText} {l.body}
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
