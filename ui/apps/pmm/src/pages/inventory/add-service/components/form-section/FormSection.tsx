import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import type { FC } from 'react';
import type { FormSectionProps } from './FormSection.types';

export const FormSection: FC<FormSectionProps> = ({
  title,
  description,
  children,
}) => (
  <Stack component="section">
    <Typography variant="h5">{title}</Typography>
    {description && (
      <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
        {description}
      </Typography>
    )}
    {children}
  </Stack>
);
