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

/** The lines before and after one line of the same node and file. */
export const LogContextDialog: FC<LogContextDialogProps> = ({
  line,
  onClose,
}) => {
  const context = useLogContext(line);
  const anchorRef = useRef<HTMLDivElement>(null);

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
          const anchor = i === context.data.anchorIndex;
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
