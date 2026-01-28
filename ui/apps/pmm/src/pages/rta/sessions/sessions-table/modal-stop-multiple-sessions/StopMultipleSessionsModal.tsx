import Button from '@mui/material/Button';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import { Dialog, DialogTitle } from '@percona/ui-lib';
import { FC, useState } from 'react';
import { Messages } from './StopMultipleSessionsModal.messages';

interface Props {
  open: boolean;
  onClose: () => void;
  onStopSessions: () => Promise<void>;
}

const StopMultipleSessionsModal: FC<Props> = ({
  open,
  onClose,
  onStopSessions,
}) => {
  const [submitting, setSubmitting] = useState(false);

  const handleStopSession = async () => {
    setSubmitting(true);
    try {
      await onStopSessions();
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Dialog
      data-testid="stop-multiple-sessions-modal"
      aria-labelledby="stop-multiple-sessions-modal-title"
      aria-describedby="stop-multiple-sessions-modal-content"
      open={open}
      onClose={onClose}
      maxWidth="xs"
      fullWidth
    >
      <DialogTitle id="stop-multiple-sessions-modal-title" onClose={onClose}>
        {Messages.title}
      </DialogTitle>
      <DialogContent id="stop-multiple-sessions-modal-content">
        {Messages.content}
      </DialogContent>
      <DialogActions>
        <Button
          data-testid="stop-multiple-sessions-modal-cancel"
          variant="text"
          color="primary"
          onClick={onClose}
        >
          {Messages.actions.cancel}
        </Button>
        <Button
          data-testid="stop-multiple-sessions-modal-stop"
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

export default StopMultipleSessionsModal;
