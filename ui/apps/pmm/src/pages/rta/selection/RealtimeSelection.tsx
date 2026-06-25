import { FC } from 'react';
import CircularProgress from '@mui/material/CircularProgress';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { Page } from 'components/page';
import { useUser } from 'contexts/user';
import { Messages } from './RealtimeSelection.messages';
import { RealtimeSelectionViewerEmptyState } from './empty-state';
import { Navigate, useNavigate } from 'react-router-dom';
import { Messages as RtaMessages } from '../messages';
import {
  useAvailableServices,
  useRealtimeSessions,
} from 'hooks/api/useRealtime';
import { DOCS_URLS } from 'lib/constants';
import { RealtimeSession } from 'types/rta.types';
import { createRealtimeOverviewUrl } from 'utils/link.utils';
import { RealtimeSelectionForm } from '../components/selection-form';
import { ServiceType } from 'types/services.types';

export const RealtimeSelection: FC = () => {
  const { user } = useUser();
  const navigate = useNavigate();
  // TODO: Add other service types when available
  const { isLoading } = useAvailableServices([
    ServiceType.mongodb,
    ServiceType.postgresql,
  ]);
  const { data: sessions, isLoading: isLoadingSessions } =
    useRealtimeSessions();

  const handleSuccess = (sessions: RealtimeSession[]) => {
    const serviceIds = sessions.map((s) => s.serviceId);

    navigate(createRealtimeOverviewUrl(serviceIds));
  };

  if (isLoading || isLoadingSessions) {
    return (
      <Page footer={null} surface="paper">
        <Stack
          sx={{
            maxWidth: 392,
            mx: 'auto',
            py: 6,
            px: 2,
            alignItems: 'center',
            justifyContent: 'center',
            minHeight: 300,
          }}
        >
          <CircularProgress />
        </Stack>
      </Page>
    );
  }

  if (!user?.isPMMAdmin) {
    return <RealtimeSelectionViewerEmptyState />;
  }

  // If there are any sessions alredy running, redirect to the sessions page
  if (sessions?.length) {
    return <Navigate to="/rta/sessions" />;
  }

  return (
    <Page footer={null} surface="paper">
      <Stack
        gap={4}
        sx={{
          maxWidth: 392,
          mx: 'auto',
          py: 6,
          px: 2,
          alignItems: 'center',
          textAlign: 'center',
        }}
      >
        <Stack gap={1} sx={{ width: '100%' }}>
          <Typography variant="h5">{Messages.title}</Typography>
          <Typography variant="body1">{Messages.description}</Typography>
        </Stack>
        <RealtimeSelectionForm onSuccess={handleSuccess} />
        <Stack gap={1} sx={{ width: '100%' }}>
          <Typography variant="body2" color="text.secondary">
            {RtaMessages.disclaimer}
          </Typography>
          <Stack direction="row" gap={2} justifyContent="center">
            <Link
              href={DOCS_URLS.qan}
              rel="noopener noreferrer"
              target="_blank"
            >
              {Messages.documentation}
            </Link>
            <Link
              href={DOCS_URLS.forums}
              rel="noopener noreferrer"
              target="_blank"
            >
              {Messages.feedback}
            </Link>
          </Stack>
        </Stack>
      </Stack>
    </Page>
  );
};
