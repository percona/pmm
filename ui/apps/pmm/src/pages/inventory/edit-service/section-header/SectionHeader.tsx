import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import type { FC } from 'react';

interface Props {
  title: string;
  description?: string;
}

export const SectionHeader: FC<Props> = ({ title, description }) => (
  <Stack gap={0.5}>
    <Typography variant="h6">{title}</Typography>
    {description && <Typography variant="body2">{description}</Typography>}
  </Stack>
);
