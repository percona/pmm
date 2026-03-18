import {
  Alert,
  Button,
  Card,
  CardContent,
  Divider,
  FormControl,
  FormControlLabel,
  InputLabel,
  Link,
  MenuItem,
  Select,
  SelectChangeEvent,
  Stack,
  Switch,
  TextField,
  Typography,
} from '@mui/material';
import { FC, useState, useEffect, ChangeEvent, SyntheticEvent } from 'react';
import { Page } from 'components/page';
import { useAdreSettings, useUpdateAdreSettings } from 'hooks/api/useAdre';
import type { AdreSettings } from 'api/adre';
import { useSnackbar } from 'notistack';
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
  const { enqueueSnackbar } = useSnackbar();
  const { data: settings, isLoading, isError, error } = useAdreSettings();
  const updateSettings = useUpdateAdreSettings();
  const [localEnabled, setLocalEnabled] = useState(settings?.enabled ?? false);
  const [localUrl, setLocalUrl] = useState(settings?.url ?? '');
  const [localChatBackend, setLocalChatBackend] = useState<'holmesgpt' | 'holmes_agent'>(
    (settings?.chatBackend === 'holmes_agent' ? 'holmes_agent' : 'holmesgpt')
  );
  const [localChatHistoryLength, setLocalChatHistoryLength] = useState(
    settings?.chatHistoryLength ?? settings?.chat_history_length ?? 20
  );
  const [localChatPrompt, setLocalChatPrompt] = useState(
    settings?.chatPromptDisplay ?? settings?.chatPrompt ?? ''
  );
  const [localInvestigationPrompt, setLocalInvestigationPrompt] = useState(
    settings?.investigationPromptDisplay ?? settings?.investigationPrompt ?? ''
  );
  const [localAgentPrompt, setLocalAgentPrompt] = useState(
    settings?.agentPromptDisplay ?? settings?.agentPrompt ?? ''
  );
  const [localQanInsightsPrompt, setLocalQanInsightsPrompt] = useState(
    settings?.qanInsightsPromptDisplay ?? settings?.qanInsightsPrompt ?? ''
  );

  useEffect(() => {
    if (settings) {
      setLocalEnabled(settings.enabled);
      setLocalUrl(settings.url);
      setLocalChatBackend(settings.chatBackend === 'holmes_agent' ? 'holmes_agent' : 'holmesgpt');
      setLocalChatHistoryLength(settings.chatHistoryLength ?? (settings as { chat_history_length?: number }).chat_history_length ?? 20);
      setLocalChatPrompt(settings.chatPromptDisplay ?? settings.chatPrompt ?? '');
      setLocalInvestigationPrompt(settings.investigationPromptDisplay ?? settings.investigationPrompt ?? '');
      setLocalAgentPrompt(settings.agentPromptDisplay ?? settings.agentPrompt ?? '');
      setLocalQanInsightsPrompt(
        settings.qanInsightsPromptDisplay ?? settings.qanInsightsPrompt ?? settings.qan_insights_prompt_display ?? settings.qan_insights_prompt ?? ''
      );
    }
  }, [
    settings?.enabled,
    settings?.url,
    settings?.chatBackend,
    settings?.chatHistoryLength,
    settings?.chatPrompt,
    settings?.chatPromptDisplay,
    settings?.investigationPrompt,
    settings?.investigationPromptDisplay,
    settings?.agentPrompt,
    settings?.agentPromptDisplay,
    settings?.qanInsightsPrompt,
    settings?.qanInsightsPromptDisplay,
  ]);

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
      <Card variant="outlined" sx={{ maxWidth: 640 }}>
        <CardContent>
          <Stack gap={3}>
            <Typography variant="body2" color="text.secondary">
              Configure the Autonomous Database Reliability Engineer (ADRE) and
              HolmesGPT integration for AI-assisted investigations.
            </Typography>
            {isAdmin ? (
              <Stack gap={3}>
                <Stack gap={2}>
                  <Typography variant="subtitle1" fontWeight={600}>
                    Connection
                  </Typography>
                  <FormControlLabel
                    control={
                      <Switch
                        checked={localEnabled}
                        onChange={(_e: SyntheticEvent, v: boolean) => setLocalEnabled(v)}
                      />
                    }
                    label="Enable ADRE"
                  />
                  <TextField
                    label="HolmesGPT URL"
                    placeholder="http://holmesgpt:8080"
                    value={localUrl}
                    onChange={(e: ChangeEvent<HTMLInputElement>) => setLocalUrl(e.target.value)}
                    size="small"
                    fullWidth
                  />
                </Stack>
                <Divider />
                <Stack gap={2}>
                  <Typography variant="subtitle1" fontWeight={600}>
                    Chat
                  </Typography>
                  <FormControl size="small" fullWidth>
                    <InputLabel>Chat backend</InputLabel>
                    <Select
                      value={localChatBackend}
                      label="Chat backend"
                      onChange={(e: SelectChangeEvent<'holmesgpt' | 'holmes_agent'>) =>
                        setLocalChatBackend(e.target.value as 'holmesgpt' | 'holmes_agent')
                      }
                    >
                      <MenuItem value="holmesgpt">Holmes Agent (direct)</MenuItem>
                      <MenuItem value="holmes_agent">PMM Agent</MenuItem>
                    </Select>
                  </FormControl>
                  {localChatBackend === 'holmes_agent' && (
                    <TextField
                      label="Conversation history length"
                      type="number"
                      inputProps={{ min: 5, max: 100 }}
                      value={localChatHistoryLength}
                      onChange={(e: ChangeEvent<HTMLInputElement>) => setLocalChatHistoryLength(parseInt(e.target.value, 10) || 20)}
                      size="small"
                      fullWidth
                      helperText="Max messages sent to PMM Agent (5–100)"
                    />
                  )}
                </Stack>
                <Divider />
                <Stack gap={2}>
                  <Typography variant="subtitle1" fontWeight={600}>
                    Prompts
                  </Typography>
                  <TextField
                    label="Chat prompt"
                    placeholder="System prompt for Holmes Agent (chat mode)"
                    value={localChatPrompt}
                    onChange={(e: ChangeEvent<HTMLInputElement>) => setLocalChatPrompt(e.target.value)}
                    size="small"
                    fullWidth
                    multiline
                    minRows={3}
                    helperText="System prompt for Holmes Agent when talking in chat mode"
                  />
                  <TextField
                    label="Investigation prompt"
                    placeholder="System prompt for investigations"
                    value={localInvestigationPrompt}
                    onChange={(e: ChangeEvent<HTMLInputElement>) => setLocalInvestigationPrompt(e.target.value)}
                    size="small"
                    fullWidth
                    multiline
                    minRows={3}
                    helperText="System prompt for Holmes Agent in investigation mode"
                  />
                  <TextField
                    label="QAN AI Insights prompt"
                    placeholder="System prompt for QAN AI Insights (query analytics and optimization). Leave empty for default."
                    value={localQanInsightsPrompt}
                    onChange={(e: ChangeEvent<HTMLInputElement>) => setLocalQanInsightsPrompt(e.target.value)}
                    size="small"
                    fullWidth
                    multiline
                    minRows={3}
                    helperText="Used when analyzing a query from Query Analytics; leave empty for default"
                  />
                  {localChatBackend === 'holmes_agent' && (
                    <TextField
                      label="Agent prompt (PMM Agent)"
                      placeholder="System prompt for PMM Agent; empty = use built-in default"
                      value={localAgentPrompt}
                      onChange={(e: ChangeEvent<HTMLInputElement>) => setLocalAgentPrompt(e.target.value)}
                      size="small"
                      fullWidth
                      multiline
                      minRows={3}
                      helperText="System prompt when using PMM Agent; leave empty for default"
                    />
                  )}
                </Stack>
                <Button
                  variant="contained"
                  onClick={() =>
                    updateSettings.mutate(
                      {
                        enabled: localEnabled,
                        url: localUrl,
                        chat_backend: localChatBackend,
                        chat_history_length: localChatHistoryLength,
                        chat_prompt: localChatPrompt || undefined,
                        investigation_prompt: localInvestigationPrompt || undefined,
                        agent_prompt: localAgentPrompt || undefined,
                        qan_insights_prompt: localQanInsightsPrompt || undefined,
                      } as Partial<AdreSettings> & Record<string, unknown>,
                      {
                        onError: (err: unknown) => {
                          const msg =
                            (err as { response?: { data?: { error?: string } } })?.response?.data
                              ?.error ??
                            (err as Error)?.message ??
                            'Failed to save settings';
                          enqueueSnackbar(msg, { variant: 'error' });
                        },
                        onSuccess: () => {
                          enqueueSnackbar('Settings saved', { variant: 'success' });
                        },
                      }
                    )
                  }
                  disabled={updateSettings.isPending}
                >
                  {updateSettings.isPending ? 'Saving...' : 'Save'}
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
