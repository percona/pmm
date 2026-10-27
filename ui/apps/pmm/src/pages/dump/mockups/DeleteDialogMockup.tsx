import Button from '@mui/material/Button';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import List from '@mui/material/List';
import ListItem from '@mui/material/ListItem';
import ListItemText from '@mui/material/ListItemText';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { Dialog, DialogTitle } from '@percona/peak-ui';
import type { FC } from 'react';
import type { Dump } from 'types/dump.types';
import { formatWindow } from './DumpCells';

interface DeleteDialogMockupProps {
  open: boolean;
  dumps: Dump[];
  onClose: () => void;
}

const describeServices = (services: string[]) =>
  services.length ? services.join(', ') : 'All services';

export const DeleteDialogMockup: FC<DeleteDialogMockupProps> = ({
  open,
  dumps,
  onClose,
}) => {
  const noun = dumps.length === 1 ? 'dataset' : `${dumps.length} datasets`;

  return (
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="sm">
      <DialogTitle onClose={onClose}>Delete {noun}?</DialogTitle>
      <DialogContent>
        <Stack gap={2}>
          <Typography>
            {dumps.length === 1 ? 'This archive is' : 'These archives are'}{' '}
            removed from PMM Server for good. Copies you already downloaded or
            sent to Support stay where they are.
          </Typography>
          <List
            dense
            disablePadding
            sx={{
              border: 1,
              borderColor: 'divider',
              borderRadius: 1,
              maxHeight: 200,
              overflowY: 'auto',
            }}
          >
            {dumps.map((dump) => (
              <ListItem key={dump.dumpId} divider>
                <ListItemText
                  primary={formatWindow(dump)}
                  secondary={describeServices(dump.serviceNames)}
                  slotProps={{ secondary: { noWrap: true } }}
                />
              </ListItem>
            ))}
          </List>
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button
          variant="contained"
          color="error"
          onClick={onClose}
          // Peak's contained error button fails contrast (dark red on red)
          sx={{ color: 'common.white' }}
        >
          Delete {noun}
        </Button>
      </DialogActions>
    </Dialog>
  );
};
