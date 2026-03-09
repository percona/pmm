import { Alert, Box, Button, Card, CardContent, FormControlLabel, Link, Stack, Switch, TextField, Typography } from '@mui/material';
import { FC, useState, useEffect } from 'react';
import { Page } from 'components/page';
import { useAdreSettings, useUpdateAdreSettings } from 'hooks/api/useAdre';
import { useUser } from 'contexts/user';
import { PMM_SETTINGS_URL } from 'lib/constants';
import { AdreChatPanel } from './components/AdreChatPanel';
import { AdreAlertsPanel } from './components/AdreAlertsPanel';

/** True when the error is an HTTP 403 (assumes axios-style error.response.status). */
function isForbiddenError(err: unknown): boolean {
  return typeof err === 'object' && err != null && 'response' in err &&
    (err as { response?: { status?: number } }).response?.status === 403;
}

const AdrePage: FC = () => {
  const { user } = useUser();
  const { data: settings, isLoading, isError, error } = useAdreSettings();
  const updateSettings = useUpdateAdreSettings();
  const [localEnabled, setLocalEnabled] = useState(settings?.enabled ?? false);
  const [localUrl, setLocalUrl] = useState(settings?.url ?? '');
  useEffect(() => {
    if (settings) {
      setLocalEnabled(settings.enabled);
      setLocalUrl(settings.url);
    }
  }, [settings?.enabled, settings?.url]);

  const isConfigured = settings?.enabled && !!settings?.url;
  const isAdmin = user?.isPMMAdmin ?? false;
  // Assumes axios-style error: error.response.status (from useQuery/useAdreSettings)
  const isForbidden = isError && isForbiddenError(error);

  if (isLoading) {
    return (
      <Page title="Autonomous Database Reliability Engineer">
        <Typography>Loading...</Typography>
      </Page>
    );
  }

  if (isError && !isForbidden) {
    return (
      <Page title="Autonomous Database Reliability Engineer">
        <Card variant="outlined">
          <CardContent>
            <Alert severity="error">
              Failed to load ADRE settings. Please try again later.
            </Alert>
          </CardContent>
        </Card>
      </Page>
    );
  }

  if (isForbidden) {
    return (
      <Page title="Autonomous Database Reliability Engineer">
        <Card variant="outlined">
          <CardContent>
            <Alert severity="info">
              Contact an administrator to configure the Autonomous Database Reliability
              Engineer (ADRE) in PMM Settings.
            </Alert>
            <Link href={PMM_SETTINGS_URL} sx={{ mt: 1, display: 'inline-block' }}>
              Open PMM Settings
            </Link>
          </CardContent>
        </Card>
      </Page>
    );
  }

  if (!isConfigured) {
    return (
      <Page title="Autonomous Database Reliability Engineer">
        <Card variant="outlined">
          <CardContent>
            <Stack gap={2}>
              <Alert severity="info">
                Configure HolmesGPT in Settings to enable the Autonomous Database
                Reliability Engineer (ADRE). Set the HolmesGPT base URL and
                enable the feature.
              </Alert>
              {isAdmin && (
                <Stack gap={2} maxWidth={480}>
                  <Typography variant="subtitle2">
                    ADRE Settings (admin only)
                  </Typography>
                  <FormControlLabel
                    control={
                      <Switch
                        checked={localEnabled}
                        onChange={(_, v) => setLocalEnabled(v)}
                      />
                    }
                    label="Enable ADRE"
                  />
                  <TextField
                    label="HolmesGPT URL"
                    placeholder="http://holmesgpt:8080"
                    value={localUrl}
                    onChange={(e) => setLocalUrl(e.target.value)}
                    size="small"
                    fullWidth
                  />
                  <Button
                    variant="contained"
                    onClick={() =>
                      updateSettings.mutate({
                        enabled: localEnabled,
                        url: localUrl,
                      })
                    }
                    disabled={updateSettings.isPending}
                  >
                    Save
                  </Button>
                </Stack>
              )}
              {!isAdmin && (
                <Link href={PMM_SETTINGS_URL}>
                  Open PMM Settings to configure ADRE
                </Link>
              )}
            </Stack>
          </CardContent>
        </Card>
      </Page>
    );
  }

  return (
    <Page title="Autonomous Database Reliability Engineer">
      <Box
        sx={{
          bgcolor: '#000',
          color: 'common.white',
          flex: 1,
          minHeight: 0,
          m: -2,
          mt: -3,
          mb: -3,
          p: 3,
          '& .MuiCard-root': {
            bgcolor: '#000',
            borderColor: 'rgba(255,255,255,0.12)',
            color: 'common.white',
          },
          '& .MuiCardContent-root': { bgcolor: 'transparent' },
          '& #messages-container': { bgcolor: '#0a0a0a' },
        }}
      >
        <Stack direction={{ xs: 'column', md: 'row' }} gap={2} sx={{ flex: 1, minHeight: 0 }}>
          <Box sx={{ flex: 1, minWidth: 0 }}>
            <AdreChatPanel />
          </Box>
          <Box sx={{ flex: 1, minWidth: 0 }}>
            <AdreAlertsPanel />
          </Box>
        </Stack>
      </Stack>
      </Box>
    </Page>
  );
};

export default AdrePage;
