import Button from '@mui/material/Button';
import IconButton from '@mui/material/IconButton';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined';
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
      maxWidth={600}
    >
      <Stack direction="row" alignItems="center" gap={1}>
        <Typography variant="body1" fontWeight={500}>
          {label}
        </Typography>
        <Tooltip
          title={
            <Stack gap={0.5}>
              <Typography variant="body2">{tooltip}</Typography>
              <Link
                href={link}
                target="_blank"
                rel="noopener noreferrer"
                color="inherit"
                sx={{ textDecoration: 'underline' }}
              >
                {tooltipLinkText}
              </Link>
            </Stack>
          }
          arrow
        >
          <IconButton size="small" aria-label={tooltip} sx={{ p: 0.5 }}>
            <InfoOutlinedIcon fontSize="small" />
          </IconButton>
        </Tooltip>
      </Stack>

      <TextField
        {...register('sshKey')}
        multiline
        minRows={4}
        placeholder="ssh-rsa AAAA..."
        fullWidth
        data-testid="ssh-key-input"
        disabled={isPending}
      />

      <Button
        type="submit"
        variant="contained"
        disabled={!isDirty || isPending}
        data-testid="ssh-key-submit"
      >
        {isPending ? 'Applying...' : action}
      </Button>
    </Stack>
  );
};
