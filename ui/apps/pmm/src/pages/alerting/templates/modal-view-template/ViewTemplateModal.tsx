import { FC } from 'react';
import Button from '@mui/material/Button';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import TextField from '@mui/material/TextField';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import { Dialog, DialogTitle } from '@percona/percona-ui';
import { enqueueSnackbar } from 'notistack';
import { copyToClipboard } from 'utils/clipboard.utils';
import { Messages } from './ViewTemplateModal.messages';
import { Messages as ListMessages } from '../AlertTemplates.messages';
import { ViewTemplateModalProps } from './ViewTemplateModal.types';

export const ViewTemplateModal: FC<ViewTemplateModalProps> = ({
  open,
  template,
  onClose,
}) => {
  if (!template) {
    return null;
  }

  const handleCopy = async () => {
    const copied = await copyToClipboard(template.yaml);
    enqueueSnackbar(
      copied ? ListMessages.copy.success : ListMessages.copy.error,
      { variant: copied ? 'success' : 'error' }
    );
  };

  return (
    <Dialog
      data-testid="view-template-modal"
      open={open}
      onClose={onClose}
      maxWidth="sm"
      fullWidth
    >
      <DialogTitle onClose={onClose}>
        {Messages.title(template.summary || template.name)}
      </DialogTitle>
      <DialogContent>
        <TextField
          id="view-template-yaml-field"
          label={Messages.yamlLabel}
          value={template.yaml}
          multiline
          minRows={10}
          fullWidth
          slotProps={{
            input: { readOnly: true },
            htmlInput: { 'data-testid': 'view-template-yaml' },
          }}
        />
      </DialogContent>
      <DialogActions>
        <Button
          variant="text"
          data-testid="view-template-close"
          onClick={onClose}
        >
          {Messages.close}
        </Button>
        <Button
          variant="contained"
          startIcon={<ContentCopyIcon />}
          data-testid="view-template-copy"
          onClick={handleCopy}
        >
          {ListMessages.actions.copy}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default ViewTemplateModal;
