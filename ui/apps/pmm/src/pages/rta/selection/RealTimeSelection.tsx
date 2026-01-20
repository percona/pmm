import { FC } from 'react';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { Page } from 'components/page';
import { useUser } from 'contexts/user';
import { useRunningRealtimeAgents } from 'hooks/api/useRealtime';
import { useServices } from 'hooks/api/useServices';
import { Messages } from './RealTimeSelection.messages';
import { ServiceType } from 'types/services.types';
import { RealTimeSelectionForm } from './RealTimeSelectionForm';
import { RealTimeSelectionEmptyState } from './RealTimeSelectionEmptyState';
import { RealTimeSelectionViewerEmptyState } from './RealTimeSelectionViewerEmptyState';
import { DOCS_URL, FORUM_URL } from './RealTimeSelection.constants';

export const RealTimeSelection: FC = () => {
  const { user } = useUser();
  const canManageRTA = user?.isEditor || user?.isPMMAdmin;

  // Fetch running agents
  const { data: runningAgentsData, isLoading: isLoadingAgents } = useRunningRealtimeAgents();

  // Fetch services to check if all are running
  const { data: servicesData, isLoading: isLoadingServices } = useServices({
    serviceType: ServiceType.mongodb,
  });

  // Check if all services are running (no available services to start)
  const runningServiceIds =
    runningAgentsData?.agents?.map((agent) => agent.serviceId) ?? [];

  const filteredServices =
    servicesData?.services?.filter(
      (service) => !runningServiceIds.includes(service.serviceId)
    ) ?? [];

  const allServicesRunning =
    !isLoadingServices &&
    !isLoadingAgents &&
    filteredServices.length === 0 &&
    servicesData?.services &&
    servicesData.services.length > 0;

  // Viewer with no running agents - show viewer empty state
  const showViewerEmptyState =
    !canManageRTA &&
    (!runningAgentsData?.agents || runningAgentsData.agents.length === 0) &&
    !isLoadingAgents;

  // Viewer with no running agents - show special empty state
  if (showViewerEmptyState) {
    return <RealTimeSelectionViewerEmptyState />;
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
          /* All services running - show empty state */
          <RealTimeSelectionEmptyState />
        ) : (
          <>
            {/* Intro section */}
            <Stack gap={1} sx={{ width: '100%' }}>
              <Typography variant="h5">
                {Messages.title}
              </Typography>
              <Typography variant="body1">
                {Messages.description}
              </Typography>
            </Stack>

            {/* Form section - reusable component */}
            <RealTimeSelectionForm />

            {/* Footer section */}
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
