import { FC } from 'react';
import CircularProgress from '@mui/material/CircularProgress';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { Page } from 'components/page';
import { useUser } from 'contexts/user';
import { Messages } from './RealtimeSelection.messages';
import { RealtimeSelectionForm } from './form/RealtimeSelectionForm';
import {
  RealtimeSelectionEmptyState,
  RealtimeSelectionViewerEmptyState,
} from './empty-state';
import { Navigate, useNavigate } from 'react-router-dom';
import { useAvailableServices } from 'hooks/api/useRealtime';
import { DOCS_URLS } from 'lib/constants';

export const RealtimeSelection: FC = () => {
  const { user } = useUser();
  const navigate = useNavigate();
  const { availableServices, isLoading, services, sessions } =
    useAvailableServices();

  const allServicesRunning =
    !isLoading &&
    availableServices.length === 0 &&
    services &&
    services.length > 0;

  const handleSuccess = () => {
    navigate('/rta/sessions');
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

  if (sessions?.length) {
    // todo: navigate to session analysis page
    return <Navigate to="/rta/sessions" />;
  }

  if (user?.isViewer) {
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
                {Messages.mongoOnly}
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
