import Alert from '@mui/material/Alert';
import AlertTitle from '@mui/material/AlertTitle';
import Button from '@mui/material/Button';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
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
  clusterChanged: boolean;
  credentialsChanged: boolean;
  dropsCredentials: boolean;
  onConfirm: () => Promise<void>;
  onClose: () => void;
}

export const ConfirmChangesModal: FC<Props> = ({
  open,
  serviceName,
  clusterChanged,
  credentialsChanged,
  dropsCredentials,
  onConfirm,
  onClose,
}) => {
  const [submitting, setSubmitting] = useState(false);

  const handleConfirm = async () => {
    setSubmitting(true);
    try {
      await onConfirm();
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Dialog
      data-testid="confirm-changes-modal"
      aria-labelledby="confirm-changes-modal-title"
      open={open}
      onClose={onClose}
      maxWidth="sm"
      fullWidth
    >
      <DialogTitle id="confirm-changes-modal-title" onClose={onClose}>
        {Messages.title(serviceName)}
      </DialogTitle>
      <DialogContent>
        <Stack gap={2}>
          <Typography>
            {confirm.description}
            <DocsLink text={confirm.docsLink} />.
          </Typography>
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
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button variant="text" onClick={onClose}>
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
