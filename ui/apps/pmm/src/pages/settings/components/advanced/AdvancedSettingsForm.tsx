import { Typography } from '@mui/material';
import { FC } from 'react';
import { Settings } from 'types/settings.types';

interface AdvancedSettingsFormProps {
  settings: Settings;
}

export const AdvancedSettingsForm: FC<AdvancedSettingsFormProps> = () => {
  return <Typography>Advanced Settings form placeholder</Typography>;
};
