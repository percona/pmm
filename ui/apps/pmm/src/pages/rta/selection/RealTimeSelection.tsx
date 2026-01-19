import { FC, useState } from 'react';
import Button from '@mui/material/Button';
import Dialog from '@mui/material/Dialog';
import DialogContent from '@mui/material/DialogContent';
import DialogTitle from '@mui/material/DialogTitle';
import IconButton from '@mui/material/IconButton';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import CloseIcon from '@mui/icons-material/Close';
import AddIcon from '@mui/icons-material/Add';
import StopIcon from '@mui/icons-material/Stop';
import { useMutation } from '@tanstack/react-query';
import { enqueueSnackbar } from 'notistack';
import { Page } from 'components/page';
import { useUser } from 'contexts/user';
import { useRunningRealtimeAgents } from 'hooks/api/useRealtime';
import { useServices } from 'hooks/api/useServices';
import { Messages } from './RealTimeSelection.messages';
import { listRunningRealtimeAgents, changeRealtimeAnalytics } from 'api/realtime';
import { ServiceType } from 'types/services.types';
import { RealTimeSelectionForm } from './RealTimeSelectionForm';
import { RealTimeSelectionEmptyState } from './RealTimeSelectionEmptyState';
import { RealTimeSelectionViewerEmptyState } from './RealTimeSelectionViewerEmptyState';
import { DOCS_URL, FORUM_URL, linkStyles } from './RealTimeSelection.constants';

// Set to true to use mock data for development/testing
// Set to false to use real MongoDB services from PMM
const USE_MOCK_DATA = false;

export const RealTimeSelection: FC = () => {
  const { user } = useUser();
  const canManageRTA = user?.isEditor || user?.isPMMAdmin;
  const [modalOpen, setModalOpen] = useState(false);

  // Fetch running agents
  const { data: runningAgentsData, isLoading: isLoadingAgents, refetch: refetchAgents } = useRunningRealtimeAgents();

  // Fetch services to check if all are running
  const { data: servicesData, isLoading: isLoadingServices } = useServices({
    serviceType: ServiceType.mongodb,
  });

  // Mutation to stop all RTA sessions
  const stopAllMutation = useMutation({
    mutationFn: async () => {
      const agentsResponse = await listRunningRealtimeAgents();

      if (!agentsResponse?.agents || agentsResponse.agents.length === 0) {
        throw new Error('No running sessions to stop');
      }

      await Promise.all(
        agentsResponse.agents.map((agent) =>
          changeRealtimeAnalytics({
            enable: false,
            serviceId: agent.serviceId,
          })
        )
      );
    },
    onSuccess: () => {
      enqueueSnackbar('All sessions stopped successfully', { variant: 'success' });
      refetchAgents();
    },
    onError: (error) => {
      const message = error instanceof Error ? error.message : 'Failed to stop sessions';
      enqueueSnackbar(message, { variant: 'error' });
    },
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
              <Typography
                variant="h5"
                sx={{
                  fontFamily: 'Poppins, sans-serif',
                  fontWeight: 600,
                  fontSize: '23px',
                  lineHeight: 1.125,
                  textAlign: 'center',
                }}
              >
                {Messages.title}
              </Typography>
              <Typography
                variant="body1"
                sx={{
                  fontFamily: 'Roboto, sans-serif',
                  fontWeight: 400,
                  fontSize: '16px',
                  lineHeight: 1.375,
                  textAlign: 'center',
                  fontVariationSettings: "'wdth' 100",
                }}
              >
                {Messages.description}
              </Typography>
            </Stack>

            {/* Form section - reusable component */}
            <RealTimeSelectionForm useMockData={USE_MOCK_DATA} />

            {/* Footer section */}
            <Stack gap={1} sx={{ width: '100%' }}>
              <Typography
                variant="body2"
                color="text.secondary"
                sx={{
                  fontFamily: 'Roboto, sans-serif',
                  fontWeight: 400,
                  fontSize: '14px',
                  lineHeight: 1.5,
                  textAlign: 'center',
                  fontVariationSettings: "'wdth' 100",
                }}
              >
                {Messages.mongoOnly}
              </Typography>
              <Stack direction="row" gap={2} justifyContent="center">
                <Link href={DOCS_URL} target="_blank" sx={linkStyles}>
                  {Messages.documentation}
                </Link>
                <Link href={FORUM_URL} target="_blank" sx={linkStyles}>
                  {Messages.feedback}
                </Link>
              </Stack>
            </Stack>
          </>
        )}

        {/* ========================================
            DEV-ONLY TEST BUTTONS
            ========================================
            These buttons are only visible in development mode for testing.
            They will not appear in production builds.
        */}
        {import.meta.env.DEV && (
          <Stack direction="row" gap={2} sx={{ mt: 4 }}>
            <Button
              variant="outlined"
              startIcon={<AddIcon />}
              onClick={() => setModalOpen(true)}
              sx={{
                borderRadius: '4px',
                textTransform: 'none',
                fontFamily: 'Roboto, sans-serif',
                fontSize: '14px',
                fontWeight: 500,
              }}
            >
              New session
            </Button>
            <Button
              variant="outlined"
              color="error"
              startIcon={<StopIcon />}
              onClick={() => stopAllMutation.mutate()}
              disabled={stopAllMutation.isPending}
              sx={{
                borderRadius: '4px',
                textTransform: 'none',
                fontFamily: 'Roboto, sans-serif',
                fontSize: '14px',
                fontWeight: 500,
              }}
            >
              {stopAllMutation.isPending ? 'Stopping...' : 'End All Sessions'}
            </Button>
          </Stack>
        )}
        {/* ======================================== */}
      </Stack>

      {/* Modal for testing reusable form component */}
      <Dialog
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        maxWidth={false}
        PaperProps={{
          sx: (theme) => ({
            borderRadius: '8px',
            width: '403px',
            maxWidth: '403px',
            height: '422px',
            backgroundColor: theme.palette.mode === 'dark'
              ? '#2C323E'  // surfaces/elevation1 - brand/stone/900
              : theme.palette.background.paper,
            backgroundImage: 'none !important',
            boxShadow: '0px 0px 1px 0px rgba(0,0,0,0.08), 0px 24px 12px 0px rgba(0,0,0,0.04), 0px 24px 48px 0px rgba(0,0,0,0.16)',
            border: '1px solid',
            borderColor: 'divider',
          }),
        }}
      >
        <DialogTitle
          sx={{
            fontFamily: 'Poppins, sans-serif',
            fontSize: '19px',
            fontWeight: 600,
            lineHeight: 1.25,
            pb: '24px',
            pt: '16px',
            px: '16px',
            pr: '48px',
          }}
        >
          Start a new session
          <IconButton
            onClick={() => setModalOpen(false)}
            sx={{
              position: 'absolute',
              right: 8,
              top: 8,
              '&:hover': {
                backgroundColor: 'action.hover',
              },
            }}
          >
            <CloseIcon />
          </IconButton>
        </DialogTitle>
        <DialogContent
          sx={{
            p: '32px',
            pt: 0,
          }}
        >
          <Stack gap="32px" alignItems="center" width="100%">
            {/* Intro */}
            <Typography
              variant="body1"
              sx={{
                fontFamily: 'Roboto, sans-serif',
                fontSize: '16px',
                fontWeight: 400,
                lineHeight: 1.375,
                textAlign: 'center',
                fontVariationSettings: "'wdth' 100",
                width: '100%',
              }}
            >
              Select a service to monitor queries and performance metrics in real time.
            </Typography>

            {/* Form */}
            <RealTimeSelectionForm
              useMockData={USE_MOCK_DATA}
              onSuccess={() => {
                setModalOpen(false);
                // TODO: Refresh agents list when this is used on "Agents Running" page
              }}
            />

            {/* Footer */}
            <Stack gap="8px" alignItems="center" width="100%">
              <Typography
                variant="body2"
                color="text.secondary"
                sx={{
                  fontFamily: 'Roboto, sans-serif',
                  fontWeight: 400,
                  fontSize: '14px',
                  lineHeight: 1.5,
                  textAlign: 'center',
                  fontVariationSettings: "'wdth' 100",
                  width: '100%',
                }}
              >
                {Messages.mongoOnly}
              </Typography>

              <Stack direction="row" gap="16px" justifyContent="center">
                <Link href={DOCS_URL} target="_blank" sx={linkStyles}>
                  Documentation
                </Link>
                <Link href={FORUM_URL} target="_blank" sx={linkStyles}>
                  Provide feedback
                </Link>
              </Stack>
            </Stack>
          </Stack>
        </DialogContent>
      </Dialog>
    </Page>
  );
};
