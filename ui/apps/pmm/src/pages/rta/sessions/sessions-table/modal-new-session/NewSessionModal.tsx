import DialogContent from '@mui/material/DialogContent';
import Typography from '@mui/material/Typography';
import { Dialog, DialogTitle } from '@percona/ui-lib';
import { FC } from 'react';
import { Messages } from './NewSessionModal.messages';
import Stack from '@mui/material/Stack';
import Link from '@mui/material/Link';
import { RealTimeSelectionForm } from 'pages/rta/selection/form/RealTimeSelectionForm';
import { DOCS_URLS } from 'lib/constants';

interface Props {
  open: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

const NewSessionModal: FC<Props> = ({ open, onClose, onSuccess }) => <Dialog open={open} onClose={onClose} >
  <DialogTitle onClose={onClose}>{Messages.title}</DialogTitle>
  <DialogContent sx={{ display: 'flex', flexDirection: 'column', gap: 4, p: 4, width: 426 }}>
    <Typography textAlign="center">{Messages.content}</Typography>
    <RealTimeSelectionForm onSuccess={onSuccess} />
    <Stack direction="column" gap={1}>
      <Typography variant="body2" color="text.secondary" textAlign="center">
        {Messages.disclaimer}
      </Typography>
      <Stack justifyContent="center" flexDirection="row" gap={2}>
        <Link
          variant="body2"
          href={DOCS_URLS.qan}
          rel="noopener noreferrer"
          target="_blank"
        >
          {Messages.documentation}
        </Link>
        <Link
          variant="body2"
          href={DOCS_URLS.qan}
          rel="noopener noreferrer"
          target="_blank"
        >
          {Messages.provideFeedback}
        </Link>
      </Stack>
    </Stack>
  </DialogContent>
</Dialog>

export default NewSessionModal;
