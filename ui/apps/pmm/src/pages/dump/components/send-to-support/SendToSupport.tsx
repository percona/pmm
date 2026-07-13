import { zodResolver } from '@hookform/resolvers/zod';
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Stack,
} from '@mui/material';
import { TextInput } from '@percona/percona-ui';
import { FC } from 'react';
import { FormProvider, useForm } from 'react-hook-form';
import { enqueueSnackbar } from 'notistack';
import { useUploadDumps } from 'hooks/api/useDump';
import { Messages } from './SendToSupport.messages';
import {
  SendToSupportFormValues,
  sendToSupportSchema,
} from './SendToSupport.schema';
import {
  SEND_TO_SUPPORT_DEFAULT_VALUES,
  toUploadPayload,
} from './SendToSupport.utils';

interface SendToSupportProps {
  dumpIds: string[];
  open: boolean;
  onClose: () => void;
}

export const SendToSupport: FC<SendToSupportProps> = ({
  dumpIds,
  open,
  onClose,
}) => {
  const { mutateAsync: upload, isPending } = useUploadDumps();
  const methods = useForm<SendToSupportFormValues>({
    resolver: zodResolver(sendToSupportSchema),
    defaultValues: SEND_TO_SUPPORT_DEFAULT_VALUES,
    mode: 'onChange',
  });
  const {
    formState: { isDirty, isValid },
    handleSubmit,
    reset,
  } = methods;

  const close = () => {
    reset(SEND_TO_SUPPORT_DEFAULT_VALUES);
    onClose();
  };

  const onSubmit = async (values: SendToSupportFormValues) => {
    await upload(toUploadPayload(values, dumpIds), {
      onSuccess: () => {
        enqueueSnackbar(Messages.success, { variant: 'success' });
        close();
      },
    });
  };

  return (
    <Dialog
      open={open}
      onClose={isPending ? undefined : close}
      fullWidth
      maxWidth="sm"
      data-testid="dump-send-to-support-dialog"
    >
      <FormProvider {...methods}>
        <Stack component="form" onSubmit={handleSubmit(onSubmit)}>
          <DialogTitle>{Messages.title}</DialogTitle>
          <DialogContent>
            <Stack gap={2} pt={1}>
              <TextInput
                name="address"
                label={Messages.address}
                textFieldProps={{
                  placeholder: Messages.addressPlaceholder,
                  slotProps: {
                    htmlInput: { 'data-testid': 'dump-support-address' },
                  },
                }}
              />
              <TextInput
                name="user"
                label={Messages.user}
                textFieldProps={{
                  slotProps: {
                    htmlInput: { 'data-testid': 'dump-support-user' },
                  },
                }}
              />
              <TextInput
                name="password"
                label={Messages.password}
                textFieldProps={{
                  type: 'password',
                  slotProps: {
                    htmlInput: { 'data-testid': 'dump-support-password' },
                  },
                }}
              />
              <TextInput
                name="directory"
                label={Messages.directory}
                textFieldProps={{
                  slotProps: {
                    htmlInput: { 'data-testid': 'dump-support-directory' },
                  },
                }}
              />
            </Stack>
          </DialogContent>
          <DialogActions>
            <Button type="button" onClick={close} disabled={isPending}>
              {Messages.cancel}
            </Button>
            <Button
              type="submit"
              variant="contained"
              disabled={!isDirty || !isValid || isPending}
            >
              {isPending ? Messages.sending : Messages.send}
            </Button>
          </DialogActions>
        </Stack>
      </FormProvider>
    </Dialog>
  );
};
