import { Box, CircularProgress, Typography } from '@mui/material';
import { FC } from 'react';

export const QanDetailsLoading: FC = () => (
  <Box sx={{ display: 'flex', justifyContent: 'center', p: 3 }}>
    <CircularProgress size={28} />
  </Box>
);

export const QanDetailsError: FC<{ message?: string }> = ({ message }) => (
  <Typography color="error">{message ?? 'Failed to load data.'}</Typography>
);
