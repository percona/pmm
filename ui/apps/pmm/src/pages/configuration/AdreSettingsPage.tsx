import {
  Alert,
  Button,
  Card,
  CardContent,
  Chip,
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

function byteCount(input: string): number {
  return new TextEncoder().encode(input).length;
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
  const [localReplaceSystemPrompt, setLocalReplaceSystemPrompt] = useState(
    settings?.replaceSystemPrompt ?? settings?.replace_system_prompt ?? false
  );
  const [localServiceNowURL, setLocalServiceNowURL] = useState(
    settings?.servicenowUrl ?? settings?.servicenow_url ?? 'https://perconadev.service-now.com/api/pellc/percona_connector/create'
  );
  const [localServiceNowAPIKey, setLocalServiceNowAPIKey] = useState('');
  const [localServiceNowClientToken, setLocalServiceNowClientToken] = useState('');
  const [localDisableRunbooks, setLocalDisableRunbooks] = useState(
    settings?.disableRunbooks ?? settings?.disable_runbooks ?? false
  );
  const [localPromptMaxBytes, setLocalPromptMaxBytes] = useState(
    settings?.promptMaxBytes ?? settings?.prompt_max_bytes ?? 16 * 1024
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
      setLocalReplaceSystemPrompt(settings.replaceSystemPrompt ?? settings.replace_system_prompt ?? false);
      setLocalServiceNowURL(
        settings.servicenowUrl ?? settings.servicenow_url ?? 'https://perconadev.service-now.com/api/pellc/percona_connector/create'
      );
      setLocalDisableRunbooks(settings.disableRunbooks ?? settings.disable_runbooks ?? false);
      setLocalPromptMaxBytes(settings.promptMaxBytes ?? settings.prompt_max_bytes ?? 16 * 1024);
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
    settings?.replaceSystemPrompt,
    settings?.servicenowUrl,
    settings?.promptMaxBytes,
    settings?.prompt_max_bytes,
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
                  <FormControlLabel
                    control={
                      <Switch
                        checked={localReplaceSystemPrompt}
                        onChange={(_e: SyntheticEvent, v: boolean) => setLocalReplaceSystemPrompt(v)}
                      />
                    }
                    label="Replace Holmes system prompt"
                  />
                  <FormControlLabel
                    control={
                      <Switch
                        checked={localDisableRunbooks}
                        onChange={(_e: SyntheticEvent, v: boolean) => setLocalDisableRunbooks(v)}
                      />
                    }
                    label="Disable Runbooks in chat"
                  />
                  <Typography variant="caption" color="text.secondary" sx={{ mt: -1 }}>
                    When enabled, the AI will not fetch or execute runbooks in chat mode.
                    Investigation mode keeps runbooks available by default.
                  </Typography>
                  <Typography variant="caption" color="text.secondary" sx={{ mt: -1 }}>
                    When enabled, the PMM prompt fully replaces Holmes&apos; default system prompt.
                    When disabled, the PMM prompt is appended to Holmes&apos; default.
                  </Typography>
                  <TextField
                    label="Prompt max bytes"
                    type="number"
                    inputProps={{ min: 1024, max: 65536 }}
                    value={localPromptMaxBytes}
                    onChange={(e: ChangeEvent<HTMLInputElement>) => setLocalPromptMaxBytes(parseInt(e.target.value, 10) || 16 * 1024)}
                    size="small"
                    fullWidth
                    helperText="Allowed range: 1024–65536. Default recommended: 16384."
                  />
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
                    helperText={`System prompt for Holmes Agent when talking in chat mode (${byteCount(localChatPrompt)} / ${localPromptMaxBytes} bytes)`}
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
                    helperText={`System prompt for Holmes Agent in investigation mode (${byteCount(localInvestigationPrompt)} / ${localPromptMaxBytes} bytes)`}
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
                    helperText={`Used when analyzing a query from Query Analytics; leave empty for default (${byteCount(localQanInsightsPrompt)} / ${localPromptMaxBytes} bytes)`}
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
                      helperText={`System prompt when using PMM Agent; leave empty for default (${byteCount(localAgentPrompt)} / ${localPromptMaxBytes} bytes)`}
                    />
                  )}
                </Stack>
                <Divider />
                <Stack gap={2}>
                  <Typography variant="subtitle1" fontWeight={600}>
                    ServiceNow Integration
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    Configure ServiceNow credentials to enable creating incident tickets from investigation reports.
                    {(settings?.servicenowConfigured ?? settings?.servicenow_configured) && (
                      <Chip label="Configured" size="small" color="success" sx={{ ml: 1 }} />
                    )}
                  </Typography>
                  <TextField
                    label="ServiceNow API URL"
                    placeholder="https://yourinstance.service-now.com/api/pellc/percona_connector/create"
                    value={localServiceNowURL}
                    onChange={(e: ChangeEvent<HTMLInputElement>) => setLocalServiceNowURL(e.target.value)}
                    size="small"
                    fullWidth
                    helperText="Percona Connector endpoint on your ServiceNow instance"
                  />
                  <TextField
                    label="API Key (x-sn-apikey)"
                    type="password"
                    placeholder="Leave empty to keep existing value"
                    value={localServiceNowAPIKey}
                    onChange={(e: ChangeEvent<HTMLInputElement>) => setLocalServiceNowAPIKey(e.target.value)}
                    size="small"
                    fullWidth
                    helperText="ServiceNow API key; leave empty to keep the current value"
                  />
                  <TextField
                    label="Client Token"
                    type="password"
                    placeholder="Leave empty to keep existing value"
                    value={localServiceNowClientToken}
                    onChange={(e: ChangeEvent<HTMLInputElement>) => setLocalServiceNowClientToken(e.target.value)}
                    size="small"
                    fullWidth
                    helperText="ServiceNow client token; leave empty to keep the current value"
                  />
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
                        replace_system_prompt: localReplaceSystemPrompt,
                        disable_runbooks: localDisableRunbooks,
                        prompt_max_bytes: localPromptMaxBytes,
                        servicenow_url: localServiceNowURL || undefined,
                        ...(localServiceNowAPIKey ? { servicenow_api_key: localServiceNowAPIKey } : {}),
                        ...(localServiceNowClientToken ? { servicenow_client_token: localServiceNowClientToken } : {}),
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
