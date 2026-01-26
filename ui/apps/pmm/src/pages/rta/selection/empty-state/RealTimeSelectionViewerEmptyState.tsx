import { FC } from 'react';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { Page } from 'components/page';
import { Messages } from '../RealTimeSelection.messages';
import { DOCS_URL } from '../RealTimeSelection.constants';
import { EmptyStateMessages } from './EmptyState.messages';

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
        {EmptyStateMessages.title}
      </Typography>
      <Typography variant="body1" color="text.secondary">
        {EmptyStateMessages.description}
      </Typography>
      <Link href={DOCS_URL} target="_blank">
        {Messages.documentation}
      </Link>
    </Stack>
  </Page>
);
