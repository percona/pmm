import { FC, useEffect } from 'react';
import Button from '@mui/material/Button';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import { Dialog, DialogTitle, TextInput } from '@percona/percona-ui';
import { zodResolver } from '@hookform/resolvers/zod';
import { FormProvider, useForm } from 'react-hook-form';
import { z } from 'zod';
import { enqueueSnackbar } from 'notistack';
import { useCreateFolder } from 'hooks/api/useFolders';
import { DashboardFolder } from 'types/folders.types';
import { Messages } from '../CreateAlertFromTemplate.messages';

interface Props {
  open: boolean;
  onClose: () => void;
  onCreated: (folder: DashboardFolder) => void;
}

const schema = z.object({
  title: z
    .string()
    .trim()
    .min(1, { message: Messages.newFolderModal.nameRequired }),
});

type NewFolderFormValues = z.infer<typeof schema>;

export const NewFolderModal: FC<Props> = ({ open, onClose, onCreated }) => {
  const { mutateAsync: createFolder, isPending } = useCreateFolder();
  const methods = useForm<NewFolderFormValues>({
    resolver: zodResolver(schema),
    mode: 'onChange',
    defaultValues: { title: '' },
  });

  useEffect(() => {
    if (open) {
      methods.reset({ title: '' });
    }
  }, [open, methods]);

  const onSubmit = async (values: NewFolderFormValues) => {
    try {
      const folder = await createFolder({ title: values.title.trim() });
      onCreated(folder);
      onClose();
    } catch (error) {
      enqueueSnackbar(
        error instanceof Error ? error.message : Messages.newFolderModal.error,
        { variant: 'error' }
      );
    }
  };

  return (
    <Dialog
      data-testid="new-folder-modal"
      open={open}
      onClose={onClose}
      maxWidth="xs"
      fullWidth
      loading={isPending}
    >
      <DialogTitle onClose={onClose}>
        {Messages.newFolderModal.title}
      </DialogTitle>
      <FormProvider {...methods}>
        <form onSubmit={methods.handleSubmit(onSubmit)}>
          <DialogContent>
            <TextInput
              name="title"
              label={Messages.newFolderModal.nameLabel}
              isRequired
              textFieldProps={{
                placeholder: Messages.newFolderModal.namePlaceholder,
                slotProps: {
                  htmlInput: { 'data-testid': 'new-folder-name' },
                },
              }}
            />
          </DialogContent>
          <DialogActions>
            <Button
              type="button"
              variant="text"
              data-testid="new-folder-cancel"
              onClick={onClose}
            >
              {Messages.newFolderModal.cancel}
            </Button>
            <Button
              type="submit"
              variant="contained"
              data-testid="new-folder-create"
              disabled={!methods.formState.isValid || isPending}
            >
              {Messages.newFolderModal.create}
            </Button>
          </DialogActions>
        </form>
      </FormProvider>
    </Dialog>
  );
};

export default NewFolderModal;
