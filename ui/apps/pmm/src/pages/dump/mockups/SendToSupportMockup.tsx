import VisibilityOffOutlinedIcon from '@mui/icons-material/VisibilityOffOutlined';
import VisibilityOutlinedIcon from '@mui/icons-material/VisibilityOutlined';
import Alert from '@mui/material/Alert';
import Button from '@mui/material/Button';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import IconButton from '@mui/material/IconButton';
import InputAdornment from '@mui/material/InputAdornment';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { Dialog, DialogTitle, TextInput } from '@percona/peak-ui';
import { type FC, useState } from 'react';
import { FormProvider, useForm } from 'react-hook-form';
import type { Dump } from 'types/dump.types';
import { formatWindow } from './DumpCells';

interface SendToSupportMockupProps {
  open: boolean;
  dumps: Dump[];
  onClose: () => void;
}

export const SendToSupportMockup: FC<SendToSupportMockupProps> = ({
  open,
  dumps,
  onClose,
}) => {
  const [showPassword, setShowPassword] = useState(false);
  const methods = useForm({
    defaultValues: { address: '', user: '', password: '', directory: '' },
  });
  const encrypted = dumps.filter(({ encrypted }) => encrypted).length;
  const noun = dumps.length === 1 ? 'dataset' : `${dumps.length} datasets`;

  return (
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="sm">
      <FormProvider {...methods}>
        <DialogTitle onClose={onClose}>
          Send {noun} to Percona Support
        </DialogTitle>
        <DialogContent>
          <Stack gap={2}>
            <Alert severity="info">
              Percona Support gives you SFTP details in your support ticket.
              Enter them below to upload the archive straight from PMM Server.
            </Alert>
            <Stack gap={0.5}>
              <Typography variant="subtitle2">Sending</Typography>
              {dumps.map((dump) => (
                <Typography
                  key={dump.dumpId}
                  variant="body2"
                  color="text.secondary"
                >
                  {formatWindow(dump)} ·{' '}
                  {dump.serviceNames.length
                    ? dump.serviceNames.join(', ')
                    : 'All services'}
                </Typography>
              ))}
            </Stack>
            <TextInput
              name="address"
              label="SFTP server"
              isRequired
              textFieldProps={{
                placeholder: 'host:port',
                helperText: 'Example: sftp.example.com:22',
              }}
            />
            <Stack direction={{ xs: 'column', sm: 'row' }} gap={2}>
              <TextInput name="user" label="Username" isRequired />
              <TextInput
                name="password"
                label="Password"
                isRequired
                textFieldProps={{
                  type: showPassword ? 'text' : 'password',
                  slotProps: {
                    input: {
                      endAdornment: (
                        <InputAdornment position="end">
                          <IconButton
                            size="small"
                            aria-label={
                              showPassword ? 'Hide password' : 'Show password'
                            }
                            onClick={() => setShowPassword((value) => !value)}
                          >
                            {showPassword ? (
                              <VisibilityOffOutlinedIcon fontSize="small" />
                            ) : (
                              <VisibilityOutlinedIcon fontSize="small" />
                            )}
                          </IconButton>
                        </InputAdornment>
                      ),
                    },
                  },
                }}
              />
            </Stack>
            <TextInput
              name="directory"
              label="Directory (optional)"
              textFieldProps={{
                placeholder: '/',
                helperText: 'Leave empty to upload to the SFTP home directory.',
              }}
            />
            {!!encrypted && (
              <Alert severity="warning">
                {encrypted === dumps.length
                  ? `${dumps.length === 1 ? 'This archive is' : 'These archives are'}`
                  : `${encrypted} of these archives ${encrypted === 1 ? 'is' : 'are'}`}{' '}
                encrypted. Share the password with Support in your ticket so
                they can open it.
              </Alert>
            )}
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={onClose}>Cancel</Button>
          <Button variant="contained" onClick={onClose}>
            Send to Support
          </Button>
        </DialogActions>
      </FormProvider>
    </Dialog>
  );
};
