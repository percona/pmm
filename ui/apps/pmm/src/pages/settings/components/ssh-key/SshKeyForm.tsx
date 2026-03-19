import { Typography } from '@mui/material';
import { FC } from 'react';
import { Settings } from 'types/settings.types';

interface SshKeyFormProps {
  settings: Settings;
}

export const SshKeyForm: FC<SshKeyFormProps> = () => {
  return <Typography>SSH Key form placeholder</Typography>;
};
