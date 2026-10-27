import Button from '@mui/material/Button';
import CircularProgress from '@mui/material/CircularProgress';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import FormControlLabel from '@mui/material/FormControlLabel';
import Stack from '@mui/material/Stack';
import Switch from '@mui/material/Switch';
import Typography from '@mui/material/Typography';
import { CodeBlock, Dialog, DialogTitle } from '@percona/peak-ui';
import { type FC, useState } from 'react';
import { type Dump, DumpStatus } from 'types/dump.types';
import { formatWindow } from './DumpCells';
import { MOCK_LOGS } from './mockups.data';

interface DumpLogsMockupProps {
  open: boolean;
  dump?: Dump;
  onClose: () => void;
}

export const DumpLogsMockup: FC<DumpLogsMockupProps> = ({
  open,
  dump,
  onClose,
}) => {
  const [follow, setFollow] = useState(true);
  const isRunning = dump?.status === DumpStatus.InProgress;

  return (
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="md">
      <DialogTitle onClose={onClose}>
        <Stack>
          <span>Dataset logs</span>
          {dump && (
            <Typography variant="body2" color="text.secondary">
              {formatWindow(dump)} ·{' '}
              {dump.serviceNames.length
                ? `${dump.serviceNames.length} services`
                : 'All services'}
            </Typography>
          )}
        </Stack>
      </DialogTitle>
      <DialogContent>
        <Stack
          direction="row"
          justifyContent="space-between"
          alignItems="center"
          mb={1}
        >
          {isRunning ? (
            <Stack direction="row" gap={1} alignItems="center">
              <CircularProgress size={14} aria-hidden />
              <Typography variant="body2" color="text.secondary">
                Export running, new lines appear as they are written
              </Typography>
            </Stack>
          ) : (
            <Typography variant="body2" color="text.secondary">
              Export finished
            </Typography>
          )}
          <FormControlLabel
            control={
              <Switch
                size="small"
                checked={follow}
                onChange={(_, checked) => setFollow(checked)}
              />
            }
            label="Follow output"
            slotProps={{ typography: { variant: 'body2' } }}
          />
        </Stack>
        <CodeBlock
          content={MOCK_LOGS}
          copyable
          wrap
          sx={{ maxHeight: 420, overflowY: 'auto' }}
        />
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Close</Button>
      </DialogActions>
    </Dialog>
  );
};
