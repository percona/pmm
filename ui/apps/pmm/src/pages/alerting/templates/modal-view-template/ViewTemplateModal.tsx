import { FC } from 'react';
import Button from '@mui/material/Button';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import TextField from '@mui/material/TextField';
import { Dialog, DialogTitle } from '@percona/percona-ui';
import { Messages } from './ViewTemplateModal.messages';
import { ViewTemplateModalProps } from './ViewTemplateModal.types';

export const ViewTemplateModal: FC<ViewTemplateModalProps> = ({
  open,
  template,
  onClose,
}) => {
  if (!template) {
    return null;
  }

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
      </DialogActions>
    </Dialog>
  );
};

export default ViewTemplateModal;
