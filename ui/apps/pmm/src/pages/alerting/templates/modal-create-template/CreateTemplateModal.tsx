import { FC } from 'react';
import Button from '@mui/material/Button';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import { Dialog, DialogTitle } from '@percona/percona-ui';
import { zodResolver } from '@hookform/resolvers/zod';
import { FormProvider, useForm } from 'react-hook-form';
import { enqueueSnackbar } from 'notistack';
import { useCreateTemplate } from 'hooks/api/useAlertTemplates';
import { TemplateYamlField } from '../components/template-yaml-field';
import { Messages } from './CreateTemplateModal.messages';
import { CreateTemplateModalProps } from './CreateTemplateModal.types';
import {
  CreateTemplateFormValues,
  createTemplateSchema,
} from './CreateTemplateModal.schema';

export const CreateTemplateModal: FC<CreateTemplateModalProps> = ({
  open,
  onClose,
}) => {
  const { mutateAsync: createTemplate, isPending } = useCreateTemplate();
  const methods = useForm<CreateTemplateFormValues>({
    resolver: zodResolver(createTemplateSchema),
    defaultValues: { yaml: '' },
  });

  const handleClose = () => {
    methods.reset({ yaml: '' });
    onClose();
  };

  const onSubmit = async (values: CreateTemplateFormValues) =>
    createTemplate(
      { yaml: values.yaml },
      {
        onSuccess: () => {
          enqueueSnackbar(Messages.success, { variant: 'success' });
          handleClose();
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
      data-testid="create-template-modal"
      open={open}
      onClose={handleClose}
      maxWidth="sm"
      fullWidth
      loading={isPending}
    >
      <DialogTitle onClose={handleClose}>{Messages.title}</DialogTitle>
      <FormProvider {...methods}>
        <form onSubmit={methods.handleSubmit(onSubmit)}>
          <DialogContent>
            <TemplateYamlField
              label={Messages.yamlLabel}
              uploadLabel={Messages.upload}
              placeholder={Messages.yamlPlaceholder}
              disabled={isPending}
            />
          </DialogContent>
          <DialogActions>
            <Button
              type="button"
              variant="text"
              data-testid="create-template-cancel"
              onClick={handleClose}
            >
              {Messages.cancel}
            </Button>
            <Button
              type="submit"
              variant="contained"
              data-testid="create-template-submit"
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

export default CreateTemplateModal;
