import { Alert, Box, Button, Card, CardContent, FormControlLabel, Link, Stack, Switch, TextField, Typography } from '@mui/material';
import { FC, useState, useEffect } from 'react';
import { Page } from 'components/page';
import { HEADER_HEIGHT } from 'components/main/header/Header.constants';
import { useAdreSettings, useAdreAlerts, useUpdateAdreSettings } from 'hooks/api/useAdre';
import { useUser } from 'contexts/user';
import { PMM_SETTINGS_URL } from 'lib/constants';
import { AdreChatPanel } from './components/AdreChatPanel';
import { AdreAlertsPanel, type AlertItem } from './components/AdreAlertsPanel';

/** True when the error is an HTTP 403 (assumes axios-style error.response.status). */
function isForbiddenError(err: unknown): boolean {
  return typeof err === 'object' && err != null && 'response' in err &&
    (err as { response?: { status?: number } }).response?.status === 403;
}

const AdrePage: FC = () => {
  const { user } = useUser();
  const { data: settings, isLoading, isError, error } = useAdreSettings();
  const updateSettings = useUpdateAdreSettings();
  const { alerts } = useAdreAlerts({ enabled: !!(settings?.enabled && settings?.url) });
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
      <Page title="">
        <Typography>Loading...</Typography>
      </Page>
    );
  }

  if (isError && !isForbidden) {
    return (
      <Page title="">
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
      <Page title="">
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
      <Page title="">
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
    <Page title="" fullWidth footer={null}>
      <Box
        sx={{
          bgcolor: '#212121',
          color: 'text.primary',
          flex: 1,
          height: `calc(100dvh - ${HEADER_HEIGHT}px - 56px)`,
          maxHeight: `calc(100dvh - ${HEADER_HEIGHT}px - 56px)`,
          minHeight: 0,
          display: 'flex',
          flexDirection: 'column',
          px: { xs: 1, sm: 2 },
          py: 2,
          borderRadius: 1,
          overflow: 'hidden',
          '& .MuiCard-root': {
            bgcolor: '#212121',
            borderColor: 'rgba(255,255,255,0.12)',
            color: 'inherit',
          },
          '& .MuiCardContent-root': { bgcolor: 'transparent' },
          '& #messages-container': { bgcolor: '#1e1e1e' },
        }}
      >
        <Stack
          direction={{ xs: 'column', md: alerts.length > 0 ? 'row' : 'column' }}
          gap={2}
          sx={{
            flex: 1,
            minHeight: 0,
            alignItems: 'stretch',
            overflow: 'hidden',
          }}
        >
          <Box
            sx={{
              flex: alerts.length > 0 ? 2 : 1,
              minWidth: 0,
              minHeight: 0,
              display: 'flex',
              flexDirection: 'column',
              overflow: 'hidden',
            }}
          >
            <AdreChatPanel />
          </Box>
          {alerts.length > 0 && (
            <Box
              sx={{
                flex: '0 0 260px',
                minWidth: 0,
                minHeight: 0,
                maxHeight: '100%',
                display: 'flex',
                flexDirection: 'column',
                overflow: 'hidden',
              }}
            >
              <AdreAlertsPanel alerts={alerts as AlertItem[]} />
            </Box>
          )}
        </Stack>
      </Box>
    </Page>
  );
};

export default AdrePage;
