import { FC } from 'react';
import Box from '@mui/material/Box';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { Icon } from 'components/icon';
import { Messages } from './RealTimeSelection.messages';
import { DOCS_URL } from './RealTimeSelection.constants';

export const RealTimeSelectionEmptyState: FC = () => {
  return (
    <Stack gap={2} alignItems="center" sx={{ width: '100%', textAlign: 'center' }}>
      <Box
        sx={{
          width: 128,
          height: 128,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
        }}
      >
        <Icon
          name="real-time-database-off"
          sx={{
            width: 192,
            height: 192,
            opacity: 0.5,
          }}
        />
      </Box>
      <Stack gap={1} sx={{ width: '100%' }}>
        <Typography variant="h6">
          No active sessions now...
        </Typography>
        <Typography variant="body1" color="text.secondary">
          Real-Time Query Analytics requires real-time agent session to collect data.
          Contact your system administrator to start a session, then try again.
        </Typography>
      </Stack>
      <Link href={DOCS_URL} target="_blank">
        {Messages.documentation}
      </Link>
    </Stack>
  );
};
