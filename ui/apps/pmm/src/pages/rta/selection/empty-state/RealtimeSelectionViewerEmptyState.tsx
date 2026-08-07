import { FC } from 'react';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { Page } from 'components/page';
import { Messages } from '../RealtimeSelection.messages';
import { DOCS_URL } from '../RealtimeSelection.constants';
import { EmptyStateMessages } from './EmptyState.messages';
import { Icon } from 'components/icon';

const RealtimeSelectionViewerEmptyState: FC = () => (
  <Page footer={null}>
    <Stack
      sx={{
        flex: 1,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
      }}
    >
      <Stack
        gap={2}
        sx={{
          p: 2,
          maxWidth: 392,
          mx: 'auto',
          alignItems: 'center',
          textAlign: 'center',
        }}
      >
        <Icon
          name="real-time-database-off"
          color="primary"
          sx={{ height: 192, width: 192, marginBottom: -5 }}
        />
        <Stack gap={1}>
          <Typography variant="h6">{EmptyStateMessages.title}</Typography>
          <Typography
            variant="body1"
            color="text.secondary"
            sx={{
              textAlign: 'justify',
              textAlignLast: 'center',
            }}
          >
            {EmptyStateMessages.description}
          </Typography>
        </Stack>
        <Link href={DOCS_URL} target="_blank">
          {Messages.documentation}
        </Link>
      </Stack>
    </Stack>
  </Page>
);

export default RealtimeSelectionViewerEmptyState;
