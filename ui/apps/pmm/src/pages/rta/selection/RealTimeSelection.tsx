import { FC } from 'react';
import CircularProgress from '@mui/material/CircularProgress';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { Page } from 'components/page';
import { useUser } from 'contexts/user';
import { useAvailableServices } from 'hooks/api/useRealtime';
import { Messages } from './RealTimeSelection.messages';
import { RealTimeSelectionForm } from './RealTimeSelectionForm';
import {
  RealTimeSelectionEmptyState,
  RealTimeSelectionViewerEmptyState,
} from './empty-state';
import { DOCS_URL, FORUM_URL } from './RealTimeSelection.constants';

export const RealTimeSelection: FC = () => {
  const { user } = useUser();
  const canManageRTA = user?.isEditor || user?.isPMMAdmin;

  const {
    availableServices,
    isLoading,
    servicesData,
    runningAgentsData,
  } = useAvailableServices();

  const allServicesRunning =
    !isLoading &&
    availableServices.length === 0 &&
    servicesData?.services &&
    servicesData.services.length > 0;

  const showViewerEmptyState =
    !canManageRTA &&
    (!runningAgentsData?.agents || runningAgentsData.agents.length === 0) &&
    !isLoading;

  if (showViewerEmptyState) {
    return <RealTimeSelectionViewerEmptyState />;
  }

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
          <RealTimeSelectionEmptyState />
        ) : (
          <>
            <Stack gap={1} sx={{ width: '100%' }}>
              <Typography variant="h5">{Messages.title}</Typography>
              <Typography variant="body1">{Messages.description}</Typography>
            </Stack>

            <RealTimeSelectionForm />

            <Stack gap={1} sx={{ width: '100%' }}>
              <Typography variant="body2" color="text.secondary">
                {Messages.mongoOnly}
              </Typography>
              <Stack direction="row" gap={2} justifyContent="center">
                <Link href={DOCS_URL} target="_blank">
                  {Messages.documentation}
                </Link>
                <Link href={FORUM_URL} target="_blank">
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
