import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  FormControlLabel,
  Stack,
  Switch,
  TextField,
  Typography,
} from '@mui/material';
import { FC, useState, useEffect } from 'react';
import { Page } from 'components/page';
import { useAdreSettings, useUpdateAdreSettings } from 'hooks/api/useAdre';
import { useUser } from 'contexts/user';
import { PMM_SETTINGS_URL } from 'lib/constants';
import { Link } from '@mui/material';
import { AdreChatPanel } from './components/AdreChatPanel';
import { AdreAlertsPanel } from './components/AdreAlertsPanel';

const AdrePage: FC = () => {
  const { user } = useUser();
  const { data: settings, isLoading } = useAdreSettings();
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

  if (isLoading) {
    return (
      <Page title="Autonomous Database Reliability Engineer">
        <Typography>Loading...</Typography>
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
      <Stack direction="column" gap={2} sx={{ height: '100%', minHeight: 0 }}>
        <Stack direction={{ xs: 'column', md: 'row' }} gap={2} sx={{ flex: 1, minHeight: 0 }}>
          <Box sx={{ flex: 1, minWidth: 0 }}>
            <AdreChatPanel />
          </Box>
          <Box sx={{ flex: 1, minWidth: 0 }}>
            <AdreAlertsPanel />
          </Box>
        </Stack>
      </Stack>
    </Page>
  );
};

export default AdrePage;
