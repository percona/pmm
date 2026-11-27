import { zodResolver } from '@hookform/resolvers/zod';
import Alert from '@mui/material/Alert';
import Button from '@mui/material/Button';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import List from '@mui/material/List';
import ListItem from '@mui/material/ListItem';
import Stack from '@mui/material/Stack';
import { Dialog, DialogTitle, TextInput } from '@percona/peak-ui';
import { SecretField } from 'components/secret-field';
import { useUploadDumps } from 'hooks/api/useDump';
import { enqueueSnackbar } from 'notistack';
import type { FC } from 'react';
import { FormProvider, useForm } from 'react-hook-form';
import type { Dump } from 'types/dump.types';
import { DatasetSummary } from '../dataset-summary';
import { Messages } from './SendToSupport.messages';
import {
  type SendToSupportFormValues,
  sendToSupportSchema,
} from './SendToSupport.schema';
import { SEND_TO_SUPPORT_DEFAULT_VALUES } from './SendToSupport.utils';

interface SendToSupportProps {
  dumps: Dump[];
  open: boolean;
  onClose: () => void;
}

export const SendToSupport: FC<SendToSupportProps> = ({
  dumps,
  open,
  onClose,
}) => {
  const { mutate: upload, isPending } = useUploadDumps();
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
  const hasEncrypted = dumps.some(({ encrypted }) => encrypted);

  const close = () => {
    reset(SEND_TO_SUPPORT_DEFAULT_VALUES);
    onClose();
  };

  const onSubmit = (values: SendToSupportFormValues) =>
    upload(
      { dumpIds: dumps.map(({ dumpId }) => dumpId), sftpParameters: values },
      {
        onSuccess: () => {
          enqueueSnackbar(Messages.success, { variant: 'success' });
          close();
        },
      }
    );

  return (
    <Dialog
      open={open}
      onClose={isPending ? undefined : close}
      fullWidth
      maxWidth="sm"
      loading={isPending}
      data-testid="dump-send-to-support-dialog"
    >
      <FormProvider {...methods}>
        <Stack component="form" onSubmit={handleSubmit(onSubmit)}>
          <DialogTitle onClose={isPending ? undefined : close}>
            {Messages.title(dumps.length)}
          </DialogTitle>
          <DialogContent>
            <Stack gap={2} pt={1}>
              <Alert severity="info">{Messages.sftpInfo}</Alert>
              <List dense disablePadding>
                {dumps.map((dump) => (
                  <ListItem key={dump.dumpId} disableGutters>
                    <DatasetSummary dump={dump} />
                  </ListItem>
                ))}
              </List>
              <TextInput
                name="address"
                label={Messages.address}
                isRequired
                textFieldProps={{
                  helperText: Messages.addressHelp,
                  slotProps: {
                    htmlInput: { 'data-testid': 'dump-support-address' },
                  },
                }}
              />
              <TextInput
                name="user"
                label={Messages.user}
                isRequired
                textFieldProps={{
                  slotProps: {
                    htmlInput: { 'data-testid': 'dump-support-user' },
                  },
                }}
              />
              <SecretField
                name="password"
                label={Messages.password}
                testId="dump-support-password"
                isRequired
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
              {hasEncrypted && (
                <Alert severity="warning">{Messages.encryptedWarning}</Alert>
              )}
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
