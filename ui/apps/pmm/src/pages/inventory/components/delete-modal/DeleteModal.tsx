import Alert from '@mui/material/Alert';
import Button from '@mui/material/Button';
import Checkbox from '@mui/material/Checkbox';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import FormControlLabel from '@mui/material/FormControlLabel';
import FormHelperText from '@mui/material/FormHelperText';
import Typography from '@mui/material/Typography';
import { Dialog, DialogTitle } from '@percona/peak-ui';
import { type FC, useState } from 'react';
import { Messages } from '../../Inventory.messages';

interface Props {
  open: boolean;
  title: string;
  warning?: string;
  message: string;
  forceDescription: string;
  confirmLabel: string;
  onConfirm: (force: boolean) => Promise<void>;
  onClose: () => void;
}

export const DeleteModal: FC<Props> = ({
  open,
  title,
  warning,
  message,
  forceDescription,
  confirmLabel,
  onConfirm,
  onClose,
}) => {
  const [force, setForce] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  const handleClose = () => {
    setForce(false);
    onClose();
  };

  const handleConfirm = async () => {
    setSubmitting(true);
    try {
      await onConfirm(force);
      setForce(false);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Dialog
      data-testid="inventory-delete-modal"
      aria-labelledby="inventory-delete-modal-title"
      open={open}
      onClose={handleClose}
      maxWidth="sm"
      fullWidth
    >
      <DialogTitle id="inventory-delete-modal-title" onClose={handleClose}>
        {title}
      </DialogTitle>
      <DialogContent>
        {warning && (
          <Alert severity="warning" sx={{ mb: 2 }}>
            {warning}
          </Alert>
        )}
        <Typography data-testid="inventory-delete-modal-message">
          {message}
        </Typography>
        <FormControlLabel
          sx={{ mt: 2 }}
          label={Messages.deleteModal.forceMode}
          control={
            <Checkbox
              checked={force}
              onChange={(e) => setForce(e.target.checked)}
              data-testid="inventory-delete-modal-force"
            />
          }
        />
        <FormHelperText sx={{ mt: 0 }}>{forceDescription}</FormHelperText>
      </DialogContent>
      <DialogActions>
        <Button
          data-testid="inventory-delete-modal-cancel"
          variant="text"
          onClick={handleClose}
        >
          {Messages.deleteModal.cancel}
        </Button>
        <Button
          data-testid="inventory-delete-modal-confirm"
          variant="contained"
          color="error"
          onClick={handleConfirm}
          disabled={submitting}
        >
          {confirmLabel}
        </Button>
      </DialogActions>
    </Dialog>
  );
};
