import { FC } from 'react';
import { ClientsModalProps } from './ClientsModal.types';
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  IconButton,
  Link,
  Stack,
  Typography,
} from '@mui/material';
import { Messages } from './ClientsModal.messages';
import { PMM_HOME_URL } from 'lib/constants';
import CloseIcon from '@mui/icons-material/Close';

export const ClientsModal: FC<ClientsModalProps> = ({ isOpen, onClose }) => (
  <Dialog
    title={Messages.title}
    open={isOpen}
    onClose={onClose}
    autoFocus={false}
    data-testid="modal-clients-update-pending"
  >
    <DialogTitle
      sx={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
      }}
      component="div"
    >
      <Typography variant="h5">{Messages.title}</Typography>
      <IconButton onClick={onClose} data-testid="modal-close-button">
        <CloseIcon />
      </IconButton>
    </DialogTitle>
    <DialogContent>
      <Stack direction="column" gap={2}>
        <Typography>{Messages.description1}</Typography>
        <Typography>{Messages.description2}</Typography>
      </Stack>
    </DialogContent>
    <DialogActions>
      <Button
        variant="text"
        onClick={onClose}
        data-testid="modal-close-window-button"
      >
        {Messages.close}
      </Button>
      <Link href={PMM_HOME_URL} data-testid="modal-pmm-home-link">
        <Button variant="contained" data-testid="modal-pmm-home-button">
          {Messages.home}
        </Button>
      </Link>
    </DialogActions>
  </Dialog>
);
