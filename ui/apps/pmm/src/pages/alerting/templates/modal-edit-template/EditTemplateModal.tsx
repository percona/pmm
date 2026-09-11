import { FC, useEffect } from 'react';
import Alert from '@mui/material/Alert';
import Button from '@mui/material/Button';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import Stack from '@mui/material/Stack';
import { Dialog, DialogTitle } from '@percona/percona-ui';
import { zodResolver } from '@hookform/resolvers/zod';
import { FormProvider, useForm } from 'react-hook-form';
import { enqueueSnackbar } from 'notistack';
import { useUpdateTemplate } from 'hooks/api/useAlertTemplates';
import { TemplateYamlField } from '../components/template-yaml-field';
import { Messages } from './EditTemplateModal.messages';
import { EditTemplateModalProps } from './EditTemplateModal.types';
import {
  EditTemplateFormValues,
  editTemplateSchema,
} from './EditTemplateModal.schema';

export const EditTemplateModal: FC<EditTemplateModalProps> = ({
  open,
  template,
  onClose,
}) => {
  const { mutateAsync: updateTemplate, isPending } = useUpdateTemplate();
  const methods = useForm<EditTemplateFormValues>({
    resolver: zodResolver(editTemplateSchema),
    defaultValues: { yaml: template?.yaml ?? '' },
  });

  useEffect(() => {
    methods.reset({ yaml: template?.yaml ?? '' });
  }, [template, methods]);

  if (!template) {
    return null;
  }

  const onSubmit = async (values: EditTemplateFormValues) =>
    updateTemplate(
      { name: template.name, yaml: values.yaml },
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
      data-testid="edit-template-modal"
      open={open}
      onClose={onClose}
      maxWidth="sm"
      fullWidth
      loading={isPending}
    >
      <DialogTitle onClose={onClose}>
        {Messages.title(template.summary || template.name)}
      </DialogTitle>
      <FormProvider {...methods}>
        <form onSubmit={methods.handleSubmit(onSubmit)}>
          <DialogContent>
            <Stack gap={2}>
              <Alert severity="info">{Messages.nameNotice}</Alert>
              <TemplateYamlField
                label={Messages.yamlLabel}
                uploadLabel={Messages.upload}
                placeholder={Messages.yamlPlaceholder}
                disabled={isPending}
              />
            </Stack>
          </DialogContent>
          <DialogActions>
            <Button
              type="button"
              variant="text"
              data-testid="edit-template-cancel"
              onClick={onClose}
            >
              {Messages.cancel}
            </Button>
            <Button
              type="submit"
              variant="contained"
              data-testid="edit-template-submit"
              disabled={isPending}
            >
              {Messages.submit}
            </Button>
          </DialogActions>
        </form>
      </FormProvider>
    </Dialog>
  );
};

export default EditTemplateModal;
