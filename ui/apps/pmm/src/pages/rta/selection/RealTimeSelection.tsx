import { FC, useState } from 'react';
import {
  Button,
  Dialog,
  DialogContent,
  DialogTitle,
  IconButton,
  Link,
  Stack,
  Typography,
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import AddIcon from '@mui/icons-material/Add';
import { useQuery } from '@tanstack/react-query';
import { Page } from 'components/page';
import { useUser } from 'contexts/user';
import { Messages } from './RealTimeSelection.messages';
import { listRunningRealtimeAgents } from 'api/realtime';
import { RealTimeSelectionForm } from './RealTimeSelectionForm';

// Set to true to use mock data for development/testing
// Set to false to use real MongoDB services from PMM
const USE_MOCK_DATA = false;

export const RealTimeSelection: FC = () => {
  const { user } = useUser();
  const canManageRTA = user?.isEditor || user?.isPMMAdmin;
  const [modalOpen, setModalOpen] = useState(false);

  // Fetch running agents for viewers to filter services
  const { data: runningAgentsData, isLoading: isLoadingAgents } = useQuery({
    queryKey: ['runningRealtimeAgents'],
    queryFn: () => listRunningRealtimeAgents(),
    enabled: !canManageRTA, // Only fetch for viewers
  });

  // Viewer with no running agents - show empty state
  const showEmptyState = !canManageRTA && (!runningAgentsData?.agents || runningAgentsData.agents.length === 0) && !isLoadingAgents;

  if (showEmptyState) {
    return (
      <Page footer={<></>}>
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
            Real-Time Query Analytics requires an active real-time agent session to collect data. Please contact a system administrator to start a session for you and check again.
          </Typography>
          <Link
            href="https://docs.percona.com/percona-monitoring-and-management/3/get-started/query-analytics.html"
            target="_blank"
            sx={(theme) => ({
              fontFamily: 'Roboto, sans-serif',
              fontSize: '14px',
              fontWeight: 400,
              lineHeight: 1.5,
              color: theme.palette.info.light,
              textDecoration: 'underline solid',
              '&:hover': {
                color: theme.palette.info.main,
              },
            })}
          >
            {Messages.documentation}
          </Link>
        </Stack>
      </Page>
    );
  }

  return (
    <Page footer={<></>}>
      <Stack
        gap={2}
        sx={{
          maxWidth: 392,
          mx: 'auto',
          py: 6,
          px: 2,
          alignItems: 'center',
          textAlign: 'center',
        }}
      >
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
            <Link
              href="https://docs.percona.com/percona-monitoring-and-management/3/get-started/query-analytics.html"
              target="_blank"
              sx={(theme) => ({
                fontFamily: 'Roboto, sans-serif',
                fontSize: '14px',
                fontWeight: 400,
                lineHeight: 1.5,
                color: theme.palette.info.light,
                textAlign: 'center',
                textDecoration: 'underline solid',
                textDecorationSkipInk: 'none',
                textUnderlinePosition: 'from-font',
                fontVariationSettings: "'wdth' 100",
                '&:hover': {
                  color: theme.palette.info.main,
                },
              })}
            >
              {Messages.documentation}
            </Link>
            <Link
              href="https://forums.percona.com/c/percona-monitoring-and-management-pmm/percona-monitoring-and-management-pmm-v3"
              target="_blank"
              sx={(theme) => ({
                fontFamily: 'Roboto, sans-serif',
                fontSize: '14px',
                fontWeight: 400,
                lineHeight: 1.5,
                color: theme.palette.info.light,
                textAlign: 'center',
                textDecoration: 'underline solid',
                textDecorationSkipInk: 'none',
                textUnderlinePosition: 'from-font',
                fontVariationSettings: "'wdth' 100",
                '&:hover': {
                  color: theme.palette.info.main,
                },
              })}
            >
              {Messages.feedback}
            </Link>
          </Stack>

          {/* Test button for modal */}
          <Button
            variant="outlined"
            startIcon={<AddIcon />}
            onClick={() => setModalOpen(true)}
            sx={{
              mt: 4,
              borderRadius: '4px',
              textTransform: 'none',
              fontFamily: 'Roboto, sans-serif',
              fontSize: '14px',
              fontWeight: 500,
            }}
          >
            New session
          </Button>
        </Stack>
      </Stack>

      {/* Modal for testing reusable form component */}
      <Dialog
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        maxWidth={false}
        PaperProps={{
          sx: (theme) => ({
            borderRadius: '8px',
            width: '422px',
            maxWidth: '422px',
            height: '403px',
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
          <Stack gap="16px" alignItems="center" width="100%">
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
            <Stack gap="8px" alignItems="center" width="100%" sx={{ mt: '16px' }}>
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
                <Link
                  href="https://docs.percona.com/percona-monitoring-and-management/3/get-started/query-analytics.html"
                  target="_blank"
                  sx={(theme) => ({
                    fontFamily: 'Roboto, sans-serif',
                    fontSize: '14px',
                    fontWeight: 400,
                    lineHeight: 1.5,
                    color: theme.palette.info.light,
                    textAlign: 'center',
                    textDecoration: 'underline solid',
                    textDecorationSkipInk: 'none',
                    textUnderlinePosition: 'from-font',
                    fontVariationSettings: "'wdth' 100",
                    '&:hover': {
                      color: theme.palette.info.main,
                    },
                  })}
                >
                  Documentation
                </Link>
                <Link
                  href="https://forums.percona.com/c/percona-monitoring-and-management-pmm/percona-monitoring-and-management-pmm-v3"
                  target="_blank"
                  sx={(theme) => ({
                    fontFamily: 'Roboto, sans-serif',
                    fontSize: '14px',
                    fontWeight: 400,
                    lineHeight: 1.5,
                    color: theme.palette.info.light,
                    textAlign: 'center',
                    textDecoration: 'underline solid',
                    textDecorationSkipInk: 'none',
                    textUnderlinePosition: 'from-font',
                    fontVariationSettings: "'wdth' 100",
                    '&:hover': {
                      color: theme.palette.info.main,
                    },
                  })}
                >
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
