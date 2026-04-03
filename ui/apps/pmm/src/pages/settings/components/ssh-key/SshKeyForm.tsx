import Button from '@mui/material/Button';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import ArrowOutwardIcon from '@mui/icons-material/ArrowOutward';
import { FC, useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { enqueueSnackbar } from 'notistack';
import { useUpdateSettings } from 'hooks/api/useSettings';
import { Messages } from '../../Settings.messages';
import { SshKeyFormProps, SshKeyFormValues } from './SshKeyForm.types';

export const SshKeyForm: FC<SshKeyFormProps> = ({ settings }) => {
  const { mutateAsync: updateSettings, isPending } = useUpdateSettings();
  const {
    register,
    handleSubmit,
    reset,
    formState: { isDirty },
  } = useForm<SshKeyFormValues>({
    defaultValues: { sshKey: settings.sshKey ?? '' },
  });

  useEffect(() => {
    reset({ sshKey: settings.sshKey ?? '' });
  }, [settings.sshKey, reset]);

  const onSubmit = async (values: SshKeyFormValues) =>
    await updateSettings(
      { sshKey: values.sshKey },
      {
        onSuccess: () => {
          enqueueSnackbar(Messages.service.success, { variant: 'success' });
          reset({ sshKey: values.sshKey });
        },
        onError: (error) => {
          enqueueSnackbar(
            error instanceof Error ? error.message : Messages.unauthorized,
            { variant: 'error' }
          );
        },
      }
    );

  const { label, link, tooltip, action } = Messages.ssh;
  const { tooltipLinkText } = Messages;

  return (
    <Stack
      component="form"
      onSubmit={handleSubmit(onSubmit)}
      gap={2}
    >
      <Stack gap={1} mb={2}>
        <Stack maxWidth={640}>
          <Typography variant="h6">
            {label}
          </Typography>
          <Typography variant="body2">
            {tooltip}
            {' '}
            <Link
              href={link}
              target="_blank"
              rel="noopener noreferrer"
            >
              {tooltipLinkText}
              <ArrowOutwardIcon sx={{ fontSize: 14 }} />
            </Link>
          </Typography>
        </Stack>
        <Stack >
          <TextField
            {...register('sshKey')}
            multiline
            minRows={4}
            placeholder="ssh-rsa AAAA..."
            data-testid="ssh-key-input"
            disabled={isPending}
          />
        </Stack>
      </Stack>

      <Stack
        sx={{
          position: 'sticky',
          bottom: 0,
          py: 2,
          bgcolor: 'background.paper',
          borderTop: 1,
          borderColor: 'divider',
          mt: 'auto',
          zIndex: 1,
          boxShadow: (theme) =>
            `-8px 0 0 0 ${theme.palette.background.paper}, 30px 0 0 0 ${theme.palette.background.paper}`,
        }}
      >
        <Button
          type="submit"
          variant="contained"
          disabled={!isDirty || isPending}
          data-testid="ssh-key-submit"
          sx={{ alignSelf: 'flex-start' }}
        >
          {isPending ? 'Applying...' : action}
        </Button>
      </Stack>
    </Stack>
  );
};
