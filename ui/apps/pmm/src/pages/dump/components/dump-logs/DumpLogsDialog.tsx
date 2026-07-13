import {
  Alert,
  Box,
  Button,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Typography,
} from '@mui/material';
import { FC, useEffect, useState } from 'react';
import { enqueueSnackbar } from 'notistack';
import { useDumpLogs } from 'hooks/api/useDump';
import { DumpLogChunk } from 'types/dump.types';
import {
  LOG_CHUNK_LIMIT,
  LOG_REFETCH_INTERVAL_MS,
} from '../../DumpPage.constants';
import { Messages } from '../../DumpPage.messages';

interface DumpLogsDialogProps {
  dumpId: string | null;
  onClose: () => void;
}

export const DumpLogsDialog: FC<DumpLogsDialogProps> = ({
  dumpId,
  onClose,
}) => {
  const [offset, setOffset] = useState(0);
  const [chunks, setChunks] = useState<DumpLogChunk[]>([]);
  const { data, isError } = useDumpLogs(
    { dumpId: dumpId ?? '', offset, limit: LOG_CHUNK_LIMIT },
    {
      enabled: !!dumpId,
      refetchInterval: (query) =>
        query.state.data?.end ? false : LOG_REFETCH_INTERVAL_MS,
    }
  );

  useEffect(() => {
    setOffset(0);
    setChunks([]);
  }, [dumpId]);

  useEffect(() => {
    if (!data?.logs.length) {
      return;
    }

    setChunks((current) => {
      const ids = new Set(current.map(({ chunkId }) => chunkId));
      return [
        ...current,
        ...data.logs.filter(({ chunkId }) => !ids.has(chunkId)),
      ];
    });
    setOffset(data.logs[data.logs.length - 1].chunkId);
  }, [data]);

  const logs = chunks.map(({ data }) => data).join('\n');
  const copyLogs = async () => {
    await navigator.clipboard.writeText(logs);
    enqueueSnackbar(Messages.logs.copied, { variant: 'success' });
  };

  return (
    <Dialog
      open={!!dumpId}
      onClose={onClose}
      fullWidth
      maxWidth="md"
      data-testid="dump-logs-modal"
    >
      <DialogTitle>{Messages.logs.title(dumpId ?? '')}</DialogTitle>
      <DialogContent>
        {isError && <Alert severity="error">{Messages.logs.error}</Alert>}
        {!data?.end && chunks.length === 0 && !isError && (
          <Box display="flex" justifyContent="center" p={3}>
            <CircularProgress aria-label={Messages.logs.loading} />
          </Box>
        )}
        {data?.end && chunks.length === 0 && !isError && (
          <Typography>{Messages.logs.empty}</Typography>
        )}
        {chunks.length > 0 && (
          <Box
            component="pre"
            sx={{
              bgcolor: 'background.default',
              borderRadius: 1,
              m: 0,
              p: 2,
              maxHeight: 480,
              overflow: 'auto',
              whiteSpace: 'pre-wrap',
              fontFamily: 'Roboto Mono, monospace',
              fontSize: 12,
            }}
          >
            {logs}
          </Box>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={copyLogs} disabled={chunks.length === 0}>
          {Messages.logs.copy}
        </Button>
        <Button onClick={onClose}>{Messages.logs.close}</Button>
      </DialogActions>
    </Dialog>
  );
};
