import Stack from '@mui/material/Stack';
import { TextInput } from '@percona/percona-ui';
import { zodResolver } from '@hookform/resolvers/zod';
import { FC, useEffect } from 'react';
import { FormProvider, useForm } from 'react-hook-form';
import { enqueueSnackbar } from 'notistack';
import { useUpdateSettings } from 'hooks/api/useSettings';
import { Messages } from '../../Settings.messages';
import { SshKeyFormProps } from './SshKeyForm.types';
import { SshKeyFormValues, sshKeySchema } from './SshKeyForm.schema';
import { SettingsFieldLabel } from '../settings-field-label';
import { formControlClasses } from '@mui/material';
import { SettingsSubmitButton } from '../settings-submit-button';
import { helperTextTestId } from 'utils/mui.utils';

export const SshKeyForm: FC<SshKeyFormProps> = ({ settings }) => {
  const { mutateAsync: updateSettings, isPending } = useUpdateSettings();
  const methods = useForm<SshKeyFormValues>({
    resolver: zodResolver(sshKeySchema),
    defaultValues: { sshKey: settings.sshKey ?? '' },
  });

  useEffect(() => {
    methods.reset({ sshKey: settings.sshKey ?? '' });
  }, [settings.sshKey, methods]);

  const onSubmit = async (values: SshKeyFormValues) =>
    await updateSettings(
      { sshKey: values.sshKey },
      {
        onSuccess: () => {
          enqueueSnackbar(Messages.service.success, { variant: 'success' });
          methods.reset({ sshKey: values.sshKey });
        },
        onError: (error) => {
          enqueueSnackbar(
            error instanceof Error ? error.message : Messages.unauthorized,
            { variant: 'error' }
          );
        },
      }
    );

  const { label, link, tooltip, placeholder } = Messages.ssh;

  return (
    <FormProvider {...methods}>
      <Stack component="form" onSubmit={methods.handleSubmit(onSubmit)} gap={2}>
        <Stack
          gap={1}
          mb={2}
          sx={{
            [`.${formControlClasses.root}`]: {
              margin: 0,
            },
          }}
        >
          <SettingsFieldLabel
            label={label}
            description={tooltip}
            readMoreLink={link}
            data-testid="ssh-key-label"
          />
          <TextInput
            name="sshKey"
            textFieldProps={{
              multiline: true,
              minRows: 4,
              placeholder,
              disabled: isPending,
              slotProps: {
                htmlInput: { 'data-testid': 'ssh-key' },
              },
            }}
            formHelperTextProps={helperTextTestId(
              'ssh-key-field-error-message'
            )}
          />
        </Stack>
        <SettingsSubmitButton testId="ssh-key-button" />
      </Stack>
    </FormProvider>
  );
};
