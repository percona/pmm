import { paperClasses } from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import { FC, PropsWithChildren } from 'react';

const RealtimeTableWrapper: FC<PropsWithChildren> = ({ children }) => (
  <Stack
    sx={{
      flex: 1,
      minHeight: 0,
      [`& > .${paperClasses.root}`]: {
        flex: 1,
        display: 'flex',
        flexDirection: 'column',
        minHeight: 0,
        overflow: 'hidden',
      },
    }}
  >
    {children}
  </Stack>
);

export default RealtimeTableWrapper;
