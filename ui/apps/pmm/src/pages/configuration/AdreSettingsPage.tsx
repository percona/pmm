import {
  Alert,
  Button,
  Card,
  CardContent,
  FormControlLabel,
  Link,
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

function isForbiddenError(err: unknown): boolean {
  return (
    typeof err === 'object' &&
    err != null &&
    'response' in err &&
    (err as { response?: { status?: number } }).response?.status === 403
  );
}

const AdreSettingsPage: FC = () => {
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

  const isAdmin = user?.isPMMAdmin ?? false;
  const isForbidden = isError && isForbiddenError(error);

  if (isLoading) {
    return (
      <Page title="AI Assistant Settings">
        <Typography>Loading...</Typography>
      </Page>
    );
  }

  if (isError && !isForbidden) {
    return (
      <Page title="AI Assistant Settings">
        <Card variant="outlined">
          <CardContent>
            <Alert severity="error">
              Failed to load AI Assistant settings. Please try again later.
            </Alert>
          </CardContent>
        </Card>
      </Page>
    );
  }

  if (isForbidden) {
    return (
      <Page title="AI Assistant Settings">
        <Card variant="outlined">
          <CardContent>
            <Alert severity="info">
              Contact an administrator to configure the AI Assistant (ADRE) in
              PMM Settings.
            </Alert>
            <Link href={PMM_SETTINGS_URL} sx={{ mt: 1, display: 'inline-block' }}>
              Open PMM Settings
            </Link>
          </CardContent>
        </Card>
      </Page>
    );
  }

  return (
    <Page title="AI Assistant Settings">
      <Card variant="outlined" sx={{ maxWidth: 560 }}>
        <CardContent>
          <Stack gap={2}>
            <Typography variant="body2" color="text.secondary">
              Configure the Autonomous Database Reliability Engineer (ADRE) and
              HolmesGPT integration for AI-assisted investigations.
            </Typography>
            {isAdmin ? (
              <Stack gap={2}>
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
            ) : (
              <Alert severity="info">
                Admin access is required to modify AI Assistant settings. Contact
                your administrator or open PMM Settings.
              </Alert>
            )}
          </Stack>
        </CardContent>
      </Card>
    </Page>
  );
};

export default AdreSettingsPage;
