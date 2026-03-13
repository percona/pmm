import { FC } from 'react';
import CircularProgress from '@mui/material/CircularProgress';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { Page } from 'components/page';
import { useUser } from 'contexts/user';
import { Messages } from './RealtimeSelection.messages';
import { Messages as RtaMessages } from '../messages';
import {
  RealtimeSelectionEmptyState,
  RealtimeSelectionViewerEmptyState,
} from './empty-state';
import { useNavigate } from 'react-router-dom';
import { useAvailableServices } from 'hooks/api/useRealtime';
import { DOCS_URLS } from 'lib/constants';
import { RealtimeSession } from 'types/rta.types';
import { createRealtimeOverviewUrl } from 'utils/link.utils';
import { RealtimeSelectionForm } from '../components/selection-form';
import { ServiceType } from 'types/services.types';

export const RealtimeSelection: FC = () => {
  const { user } = useUser();
  const navigate = useNavigate();
  // TODO: Add other service types when available
  const { availableServices, isLoading, services } = useAvailableServices([ServiceType.mongodb]);

  const allServicesRunning =
    !isLoading && availableServices.length === 0 && services.mongodb.length > 0;

  const handleSuccess = (sessions: RealtimeSession[]) => {
    const serviceIds = sessions.map((s) => s.serviceId);

    navigate(createRealtimeOverviewUrl(serviceIds));
  };

  if (isLoading) {
    return (
      <Page footer={null}>
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

  return (
    <Page footer={null}>
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
        {allServicesRunning ? (
          <RealtimeSelectionEmptyState />
        ) : (
          <>
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
          </>
        )}
      </Stack>
    </Page>
  );
};
