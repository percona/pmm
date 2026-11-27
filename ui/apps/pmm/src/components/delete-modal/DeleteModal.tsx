import Alert from '@mui/material/Alert';
import Button from '@mui/material/Button';
import Checkbox from '@mui/material/Checkbox';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import FormControlLabel from '@mui/material/FormControlLabel';
import FormHelperText from '@mui/material/FormHelperText';
import Typography from '@mui/material/Typography';
import { Dialog, DialogTitle } from '@percona/peak-ui';
import { type FC, type ReactNode, useState } from 'react';
import { Messages } from './DeleteModal.messages';

interface Props {
  open: boolean;
  title: string;
  message: string;
  confirmLabel: string;
  testId: string;
  warning?: string;
  forceDescription?: string;
  children?: ReactNode;
  onConfirm: (force: boolean) => Promise<void>;
  onClose: () => void;
}

export const DeleteModal: FC<Props> = ({
  open,
  title,
  message,
  confirmLabel,
  testId,
  warning,
  forceDescription,
  children,
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
      data-testid={testId}
      aria-labelledby={`${testId}-title`}
      open={open}
      onClose={handleClose}
      maxWidth="sm"
      fullWidth
    >
      <DialogTitle
        id={`${testId}-title`}
        data-testid={`${testId}-title`}
        onClose={handleClose}
      >
        {title}
      </DialogTitle>
      <DialogContent>
        {warning && (
          <Alert severity="warning" sx={{ mb: 2 }}>
            {warning}
          </Alert>
        )}
        <Typography data-testid={`${testId}-message`}>{message}</Typography>
        {children}
        {forceDescription && (
          <>
            <FormControlLabel
              sx={{ mt: 2 }}
              label={Messages.forceMode}
              control={
                <Checkbox
                  checked={force}
                  onChange={(e) => setForce(e.target.checked)}
                  data-testid={`${testId}-force`}
                />
              }
            />
            <FormHelperText sx={{ mt: 0 }}>{forceDescription}</FormHelperText>
          </>
        )}
      </DialogContent>
      <DialogActions>
        <Button
          data-testid={`${testId}-cancel`}
          variant="text"
          onClick={handleClose}
        >
          {Messages.cancel}
        </Button>
        <Button
          data-testid={`${testId}-confirm`}
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
