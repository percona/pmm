import Box from '@mui/material/Box';
import type { FC, PropsWithChildren } from 'react';

export const FieldGrid: FC<PropsWithChildren> = ({ children }) => (
  <Box
    sx={{
      display: 'grid',
      gridTemplateColumns: { xs: '1fr', md: '1fr 1fr' },
      columnGap: 2,
    }}
  >
    {children}
  </Box>
);
