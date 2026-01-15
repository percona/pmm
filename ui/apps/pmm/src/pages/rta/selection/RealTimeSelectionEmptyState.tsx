import { FC } from 'react';
import { Stack, Typography, Link, Box } from '@mui/material';
import { Icon } from 'components/icon';
import { Messages } from './RealTimeSelection.messages';
import { DOCS_URL, linkStyles, titleStyles, descriptionStyles } from './RealTimeSelection.constants';

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
        <Typography variant="h6" sx={titleStyles}>
          No active sessions now...
        </Typography>
        <Typography variant="body1" sx={descriptionStyles}>
          Real-Time Query Analytics requires an active real-time agent session to collect data.
          Please contact a system administrator to start a session for you and check again.
        </Typography>
      </Stack>
      <Link href={DOCS_URL} target="_blank" sx={linkStyles}>
        {Messages.documentation}
      </Link>
    </Stack>
  );
};
