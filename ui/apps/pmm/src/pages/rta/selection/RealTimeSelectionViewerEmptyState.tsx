import { FC } from 'react';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { Page } from 'components/page';
import { Messages } from './RealTimeSelection.messages';
import { DOCS_URL } from './RealTimeSelection.constants';

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
      <Typography variant="h6">
        No active sessions now...
      </Typography>
      <Typography variant="body1" color="text.secondary">
        Real-Time Query Analytics requires real-time agent session to collect data.
        Contact your system administrator to start a session, then try again.
      </Typography>
      <Link href={DOCS_URL} target="_blank">
        {Messages.documentation}
      </Link>
    </Stack>
  </Page>
);
