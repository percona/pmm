import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Collapse,
  FormControl,
  FormControlLabel,
  InputLabel,
  Link,
  MenuItem,
  Select,
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
import { AdreChatPanel } from './components/AdreChatPanel';
import { AdreAlertsPanel } from './components/AdreAlertsPanel';

/** True when the error is an HTTP 403 (assumes axios-style error.response.status). */
function isForbiddenError(err: unknown): boolean {
  return typeof err === 'object' && err != null && 'response' in err &&
    (err as { response?: { status?: number } }).response?.status === 403;
}

const ADRE_PROMPT_MAX_LENGTH = 2048;

const AdrePage: FC = () => {
  const { user } = useUser();
  const { data: settings, isLoading, isError, error } = useAdreSettings();
  const updateSettings = useUpdateAdreSettings();
  const [localEnabled, setLocalEnabled] = useState(settings?.enabled ?? false);
  const [localUrl, setLocalUrl] = useState(settings?.url ?? '');
  const [localChatPrompt, setLocalChatPrompt] = useState(settings?.chatPrompt ?? '');
  const [localInvestigationPrompt, setLocalInvestigationPrompt] = useState(settings?.investigationPrompt ?? '');
  const [localDefaultChatMode, setLocalDefaultChatMode] = useState<'chat' | 'investigation'>(
    settings?.defaultChatMode === 'investigation' ? 'investigation' : 'chat'
  );
  const [promptsSectionOpen, setPromptsSectionOpen] = useState(false);
  useEffect(() => {
    if (settings) {
      setLocalEnabled(settings.enabled);
      setLocalUrl(settings.url);
      setLocalChatPrompt(settings.chatPrompt ?? '');
      setLocalInvestigationPrompt(settings.investigationPrompt ?? '');
      setLocalDefaultChatMode(settings.defaultChatMode === 'investigation' ? 'investigation' : 'chat');
    }
  }, [settings?.enabled, settings?.url, settings?.chatPrompt, settings?.investigationPrompt, settings?.defaultChatMode]);

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

  const handleSavePrompts = () => {
    if (new Blob([localChatPrompt]).size > ADRE_PROMPT_MAX_LENGTH) {
      return;
    }
    if (new Blob([localInvestigationPrompt]).size > ADRE_PROMPT_MAX_LENGTH) {
      return;
    }
    updateSettings.mutate({
      chatPrompt: localChatPrompt,
      investigationPrompt: localInvestigationPrompt,
      defaultChatMode: localDefaultChatMode,
    });
  };

  const chatPromptOver = new Blob([localChatPrompt]).size > ADRE_PROMPT_MAX_LENGTH;
  const investigationPromptOver = new Blob([localInvestigationPrompt]).size > ADRE_PROMPT_MAX_LENGTH;

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
        <Stack direction="column" gap={2} sx={{ height: '100%', minHeight: 0 }}>
          {isAdmin && (
            <Card variant="outlined">
              <CardContent>
                <Button
                  onClick={() => setPromptsSectionOpen((o) => !o)}
                  sx={{ justifyContent: 'flex-start', textTransform: 'none', width: '100%' }}
                >
                  {promptsSectionOpen ? '▼' : '▶'} ADRE behavior (prompts and default mode)
                </Button>
                <Collapse in={promptsSectionOpen}>
                  <Stack gap={2} sx={{ mt: 2, maxWidth: 720 }}>
                    <TextField
                      label="Chat prompt (fast mode)"
                      placeholder="Leave empty to use built-in default. Max 2048 characters."
                      value={localChatPrompt}
                      onChange={(e) => setLocalChatPrompt(e.target.value)}
                      multiline
                      minRows={4}
                      maxRows={12}
                      size="small"
                      fullWidth
                      error={chatPromptOver}
                      helperText={chatPromptOver ? `Max ${ADRE_PROMPT_MAX_LENGTH} characters` : undefined}
                      inputProps={{ maxLength: ADRE_PROMPT_MAX_LENGTH }}
                    />
                    <TextField
                      label="Investigation prompt"
                      placeholder="Leave empty to use built-in default. Max 2048 characters."
                      value={localInvestigationPrompt}
                      onChange={(e) => setLocalInvestigationPrompt(e.target.value)}
                      multiline
                      minRows={4}
                      maxRows={12}
                      size="small"
                      fullWidth
                      error={investigationPromptOver}
                      helperText={investigationPromptOver ? `Max ${ADRE_PROMPT_MAX_LENGTH} characters` : undefined}
                      inputProps={{ maxLength: ADRE_PROMPT_MAX_LENGTH }}
                    />
                    <FormControl size="small" sx={{ minWidth: 200 }}>
                      <InputLabel>Default chat mode</InputLabel>
                      <Select
                        value={localDefaultChatMode}
                        label="Default chat mode"
                        onChange={(e) => setLocalDefaultChatMode(e.target.value as 'chat' | 'investigation')}
                      >
                        <MenuItem value="chat">Chat (fast)</MenuItem>
                        <MenuItem value="investigation">Investigation</MenuItem>
                      </Select>
                    </FormControl>
                    <Button
                      variant="contained"
                      onClick={handleSavePrompts}
                      disabled={updateSettings.isPending || chatPromptOver || investigationPromptOver}
                    >
                      Save
                    </Button>
                  </Stack>
                </Collapse>
              </CardContent>
            </Card>
          )}
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
