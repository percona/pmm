import Alert from '@mui/material/Alert';
import AlertTitle from '@mui/material/AlertTitle';
import Button from '@mui/material/Button';
import Checkbox from '@mui/material/Checkbox';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import FormControlLabel from '@mui/material/FormControlLabel';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { Dialog, DialogTitle } from '@percona/peak-ui';
import { DOCS_URLS } from 'lib/constants';
import { type FC, useState } from 'react';
import { Messages } from '../EditService.messages';

const { confirm } = Messages;

const DocsLink: FC<{ text: string }> = ({ text }) => (
  <Link
    href={DOCS_URLS.editingLabels}
    target="_blank"
    rel="noopener noreferrer"
  >
    {text}
  </Link>
);

interface Props {
  open: boolean;
  serviceName: string;
  labelsChanged: boolean;
  settingsChanged: boolean;
  clusterChanged: boolean;
  credentialsChanged: boolean;
  dropsCredentials: boolean;
  // The connection check is skipped for this save only; the server does not
  // store the choice
  onConfirm: (skipConnectionCheck: boolean) => Promise<void>;
  onClose: () => void;
}

export const ConfirmChangesModal: FC<Props> = ({
  open,
  serviceName,
  labelsChanged,
  settingsChanged,
  clusterChanged,
  credentialsChanged,
  dropsCredentials,
  onConfirm,
  onClose,
}) => {
  const [submitting, setSubmitting] = useState(false);
  const [skipConnectionCheck, setSkipConnectionCheck] = useState(false);

  const handleClose = () => {
    setSkipConnectionCheck(false);
    onClose();
  };

  const handleConfirm = async () => {
    setSubmitting(true);
    try {
      await onConfirm(settingsChanged && skipConnectionCheck);
    } finally {
      setSubmitting(false);
      setSkipConnectionCheck(false);
    }
  };

  return (
    <Dialog
      data-testid="confirm-changes-modal"
      aria-labelledby="confirm-changes-modal-title"
      open={open}
      onClose={handleClose}
      maxWidth="sm"
      fullWidth
    >
      <DialogTitle id="confirm-changes-modal-title" onClose={handleClose}>
        {Messages.title(serviceName)}
      </DialogTitle>
      <DialogContent>
        <Stack gap={2}>
          {labelsChanged && (
            <Typography>
              {confirm.description}
              <DocsLink text={confirm.docsLink} />.
            </Typography>
          )}
          {settingsChanged && (
            <Typography data-testid="settings-changed-info">
              {confirm.settingsDescription}
            </Typography>
          )}
          {clusterChanged && (
            <Alert severity="warning" data-testid="cluster-changed-warning">
              <AlertTitle>{confirm.cluster.title}</AlertTitle>
              {confirm.cluster.description}
              <DocsLink text={confirm.cluster.docsLink} />.
            </Alert>
          )}
          {credentialsChanged && (
            <Alert severity="info" data-testid="credentials-changed-info">
              <AlertTitle>{confirm.credentials.title}</AlertTitle>
              {confirm.credentials.description}
            </Alert>
          )}
          {dropsCredentials && (
            <Alert severity="warning" data-testid="host-credentials-warning">
              <AlertTitle>{confirm.hostCredentials.title}</AlertTitle>
              {confirm.hostCredentials.description}
            </Alert>
          )}
          {settingsChanged && (
            <Stack>
              <FormControlLabel
                label={confirm.skipConnectionCheck}
                control={
                  <Checkbox
                    checked={skipConnectionCheck}
                    onChange={(event) =>
                      setSkipConnectionCheck(event.target.checked)
                    }
                    data-testid="skip-connection-check"
                  />
                }
              />
              <Typography variant="body2" color="text.secondary">
                {confirm.skipConnectionCheckDescription}
              </Typography>
            </Stack>
          )}
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button variant="text" onClick={handleClose}>
          {confirm.cancel}
        </Button>
        <Button
          variant="contained"
          onClick={handleConfirm}
          disabled={submitting}
          data-testid="confirm-changes-modal-confirm"
        >
          {confirm.confirm}
        </Button>
      </DialogActions>
    </Dialog>
  );
};
