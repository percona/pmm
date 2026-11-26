import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import CircularProgress from '@mui/material/CircularProgress';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { CodeBlock, Dialog, DialogTitle } from '@percona/peak-ui';
import { useDumpLogs } from 'hooks/api/useDump';
import { type FC, useEffect, useState } from 'react';
import type { Dump, DumpLogChunk } from 'types/dump.types';
import {
  LOG_CHUNK_LIMIT,
  LOG_REFETCH_INTERVAL_MS,
} from '../../DumpPage.constants';
import { Messages } from '../../DumpPage.messages';
import { DatasetSummary } from '../dataset-summary';

interface DumpLogsDialogProps {
  dump: Dump | null;
  open: boolean;
  onClose: () => void;
}

export const DumpLogsDialog: FC<DumpLogsDialogProps> = ({
  dump,
  open,
  onClose,
}) => {
  const [offset, setOffset] = useState(0);
  const [chunks, setChunks] = useState<DumpLogChunk[]>([]);
  const dumpId = dump?.dumpId ?? '';
  const { data, isError } = useDumpLogs(
    { dumpId, offset, limit: LOG_CHUNK_LIMIT },
    {
      enabled: open && !!dumpId,
      refetchInterval: (query) =>
        query.state.data?.end ? false : LOG_REFETCH_INTERVAL_MS,
    }
  );

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

  return (
    <Dialog
      open={open}
      onClose={onClose}
      fullWidth
      maxWidth="md"
      data-testid="dump-logs-modal"
    >
      <DialogTitle onClose={onClose}>{Messages.logs.title}</DialogTitle>
      <DialogContent>
        <Stack gap={2}>
          {dump && <DatasetSummary dump={dump} />}
          {data && (
            <Typography variant="body2" color="text.secondary">
              {data.end ? Messages.logs.finished : Messages.logs.running}
            </Typography>
          )}
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
            <CodeBlock
              content={logs}
              copyable
              wrap
              sx={{ maxHeight: 480, overflowY: 'auto' }}
            />
          )}
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{Messages.logs.close}</Button>
      </DialogActions>
    </Dialog>
  );
};
