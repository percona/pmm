import { FC, PropsWithChildren } from 'react';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import CircularProgress from '@mui/material/CircularProgress';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { Link as RouterLink } from 'react-router-dom';
import {
  PERCONA_SUPPORT_URL,
  PMM_SERVICENOW_SETTINGS_PATH,
  SUPPORT_DIAGNOSTICS_DOCS_URL,
} from 'lib/constants';
import { useServiceNowConnection } from 'pages/settings/components/servicenow/ServiceNowConnection.hooks';
import { Messages } from './ServiceNowSetupGate.messages';

/**
 * What an operator sees instead of the app while delivery is unconfigured:
 * what the tool does, where to connect it, and what it will not do unasked.
 */
const SetupPrompt: FC = () => (
  <Stack gap={3} sx={{ flex: 1 }}>
    <Typography variant="h4">{Messages.heading}</Typography>
    <Stack
      alignItems="center"
      justifyContent="center"
      sx={{ flex: 1 }}
      data-testid="servicenow-setup-prompt"
    >
      <Stack
        alignItems="center"
        textAlign="center"
        sx={{ maxWidth: 480, width: '100%', p: 2 }}
      >
        <Stack gap={1}>
          <Typography variant="h5">{Messages.title}</Typography>
          <Typography variant="body1">{Messages.description}</Typography>
        </Stack>

        <Button
          variant="contained"
          component={RouterLink}
          to={PMM_SERVICENOW_SETTINGS_PATH}
          data-testid="servicenow-setup-cta"
          sx={{ my: 4 }}
        >
          {Messages.setUp}
        </Button>

        <Stack gap={2}>
          <Typography variant="body2">
            {Messages.consentNote}{' '}
            <Link
              href={SUPPORT_DIAGNOSTICS_DOCS_URL}
              target="_blank"
              rel="noopener noreferrer"
            >
              {Messages.howItWorks}
            </Link>
          </Typography>
          <Typography variant="body2">
            {Messages.subscriptionPrompt}
            <br />
            <Link
              href={PERCONA_SUPPORT_URL}
              target="_blank"
              rel="noopener noreferrer"
            >
              {Messages.subscriptionLinkText}
            </Link>
          </Typography>
        </Stack>
      </Stack>
    </Stack>
  </Stack>
);

/**
 * Holds the Support diagnostics app until ServiceNow delivery is configured.
 *
 * Anything the app can do ends in an upload to a ServiceNow case, so an
 * unconfigured deployment has nothing to offer but a dead end — the prompt
 * replaces the app rather than sitting beside it, and points at the settings
 * tab that fixes it.
 *
 * A settings read that failed says nothing about the connection, so it does
 * *not* produce the prompt: the app renders and reports its own failures, which
 * is better than telling an operator with a working connection to go configure
 * one. `drifted` does gate — SEP holds values the current delivery plan no
 * longer accepts, so delivery is as broken as if nothing were stored.
 */
export const ServiceNowSetupGate: FC<PropsWithChildren> = ({ children }) => {
  const { status, isLoading, error } = useServiceNowConnection();

  if (isLoading) {
    return (
      <Box
        sx={{
          flex: 1,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          p: 4,
        }}
      >
        <CircularProgress aria-label={Messages.loading} />
      </Box>
    );
  }

  if (error || status === 'configured') {
    return <>{children}</>;
  }

  return <SetupPrompt />;
};
