import Button from '@mui/material/Button';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import { Dialog, DialogTitle } from '@percona/ui-lib';
import { Messages } from './StopSessionModal.messages';
import { FC, useState } from 'react';

interface Props {
  open: boolean;
  onClose: () => void;
  onStopSession: () => Promise<void>;
}

const StopSessionModal: FC<Props> = ({ open, onClose, onStopSession }) => {
  const [submitting, setSubmitting] = useState(false);

  const handleStopSession = async () => {
    setSubmitting(true);
    await onStopSession();
    setSubmitting(false);
  };

  return (
    <Dialog
      aria-labelledby="stop-session-modal-title"
      aria-describedby="stop-session-modal-content"
      open={open}
      onClose={onClose}
      maxWidth="xs"
      fullWidth
    >
      <DialogTitle id="stop-session-modal-title" onClose={onClose}>
        {Messages.title}
      </DialogTitle>
      <DialogContent id="stop-session-modal-content">
        {Messages.content}
      </DialogContent>
      <DialogActions>
        <Button variant="text" color="primary" onClick={onClose}>
          {Messages.actions.cancel}
        </Button>
        <Button
          variant="contained"
          color="primary"
          onClick={handleStopSession}
          disabled={submitting}
        >
          {Messages.actions.stop}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default StopSessionModal;
