import { FC } from 'react';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { Page } from 'components/page';
import { Messages } from './RealTimeSelection.messages';
import { DOCS_URL, linkStyles } from './RealTimeSelection.constants';

export const RealTimeSelectionViewerEmptyState: FC = () => (
  <Page footer={null}>
    <Stack
      gap={3}
      sx={{
        maxWidth: 392,
        mx: 'auto',
        py: 6,
        px: 2,
        alignItems: 'center',
        textAlign: 'center',
      }}
    >
      <Typography
        variant="h6"
        sx={{
          fontFamily: 'Poppins, sans-serif',
          fontWeight: 600,
          fontSize: '18px',
          lineHeight: 1.3,
        }}
      >
        No active sessions now...
      </Typography>
      <Typography
        variant="body1"
        color="text.secondary"
        sx={{
          fontFamily: 'Roboto, sans-serif',
          fontWeight: 400,
          fontSize: '16px',
          lineHeight: 1.5,
          maxWidth: 360,
        }}
      >
        Real-Time Query Analytics requires an active real-time agent session to collect data.
        Please contact a system administrator to start a session for you and check again.
      </Typography>
      <Link href={DOCS_URL} target="_blank" sx={linkStyles}>
        {Messages.documentation}
      </Link>
    </Stack>
  </Page>
);
