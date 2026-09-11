import { FC } from 'react';
import Button from '@mui/material/Button';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import { Dialog, DialogTitle } from '@percona/percona-ui';
import { enqueueSnackbar } from 'notistack';
import { useDeleteTemplate } from 'hooks/api/useAlertTemplates';
import { Template } from 'types/alert-templates.types';
import { Messages } from './DeleteTemplateModal.messages';

interface Props {
  open: boolean;
  template: Template | null;
  onClose: () => void;
}

export const DeleteTemplateModal: FC<Props> = ({ open, template, onClose }) => {
  const { mutateAsync: deleteTemplate, isPending } = useDeleteTemplate();

  if (!template) {
    return null;
  }

  const handleDelete = async () =>
    deleteTemplate(
      { name: template.name },
      {
        onSuccess: () => {
          enqueueSnackbar(Messages.success, { variant: 'success' });
          onClose();
        },
        onError: (error) =>
          enqueueSnackbar(
            error instanceof Error ? error.message : Messages.error,
            { variant: 'error' }
          ),
      }
    );

  return (
    <Dialog
      data-testid="delete-template-modal"
      open={open}
      onClose={onClose}
      maxWidth="xs"
      fullWidth
      loading={isPending}
    >
      <DialogTitle onClose={onClose}>{Messages.title}</DialogTitle>
      <DialogContent>
        {Messages.content(template.summary || template.name)}
      </DialogContent>
      <DialogActions>
        <Button
          variant="text"
          data-testid="delete-template-cancel"
          onClick={onClose}
        >
          {Messages.cancel}
        </Button>
        <Button
          variant="contained"
          color="error"
          data-testid="delete-template-confirm"
          disabled={isPending}
          onClick={handleDelete}
        >
          {Messages.confirm}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default DeleteTemplateModal;
