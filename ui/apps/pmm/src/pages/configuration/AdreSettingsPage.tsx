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
import { useAdreModels, useAdreSettings, useUpdateAdreSettings } from 'hooks/api/useAdre';
import type { AdreSettings } from 'api/adre';
import {
  AdreBehaviorControlsBlock,
  hydrateAdreBehaviorMap,
} from 'pages/configuration/AdreBehaviorControlsBlock';
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

function behaviorFromSettings(
  s: AdreSettings,
  camel: keyof AdreSettings,
  snake: string
): Record<string, boolean> | undefined {
  const raw = s as unknown as Record<string, unknown>;
  const v = raw[camel as string] ?? raw[snake];
  if (!v || typeof v !== 'object' || Array.isArray(v)) return undefined;
  return v as Record<string, boolean>;
}

const AdreSettingsPage: FC = () => {
  const { user } = useUser();
  const { enqueueSnackbar } = useSnackbar();
  const { data: settings, isLoading, isError, error } = useAdreSettings();
  const { data: models = [] } = useAdreModels({ enabled: true });
  const updateSettings = useUpdateAdreSettings();
  const [localEnabled, setLocalEnabled] = useState(settings?.enabled ?? false);
  const [localUrl, setLocalUrl] = useState(settings?.url ?? '');
  const [localDefaultChatMode, setLocalDefaultChatMode] = useState<'fast' | 'investigation'>('investigation');
  const [localFastModel, setLocalFastModel] = useState('');
  const [localInvestigationModel, setLocalInvestigationModel] = useState('');
  const [localQanInsightsModel, setLocalQanInsightsModel] = useState('');
  const [localAdreMaxConversationMessages, setLocalAdreMaxConversationMessages] = useState(40);
  const [localBehaviorFast, setLocalBehaviorFast] = useState<Record<string, boolean>>(() =>
    hydrateAdreBehaviorMap(undefined, 'fast')
  );
  const [localBehaviorInvestigation, setLocalBehaviorInvestigation] = useState<Record<string, boolean>>(() =>
    hydrateAdreBehaviorMap(undefined, 'investigation')
  );
  const [localBehaviorFormat, setLocalBehaviorFormat] = useState<Record<string, boolean>>(() =>
    hydrateAdreBehaviorMap(undefined, 'format')
  );
  const [localChatPrompt, setLocalChatPrompt] = useState(
    settings?.chatPromptDisplay ?? settings?.chatPrompt ?? ''
  );
  const [localInvestigationPrompt, setLocalInvestigationPrompt] = useState(
    settings?.investigationPromptDisplay ?? settings?.investigationPrompt ?? ''
  );
  const [localQanInsightsPrompt, setLocalQanInsightsPrompt] = useState(
    settings?.qanInsightsPromptDisplay ?? settings?.qanInsightsPrompt ?? ''
  );
  const [localServiceNowURL, setLocalServiceNowURL] = useState(
    settings?.servicenowUrl ?? settings?.servicenow_url ?? 'https://perconadev.service-now.com/api/pellc/percona_connector/create'
  );
  const [localServiceNowAPIKey, setLocalServiceNowAPIKey] = useState('');
  const [localServiceNowClientToken, setLocalServiceNowClientToken] = useState('');
  const [localPromptMaxBytes, setLocalPromptMaxBytes] = useState(
    settings?.promptMaxBytes ?? settings?.prompt_max_bytes ?? 16 * 1024
  );
  const [localSlackEnabled, setLocalSlackEnabled] = useState(
    settings?.slackEnabled ?? settings?.slack_enabled ?? false
  );
  const [localSlackBotToken, setLocalSlackBotToken] = useState('');
  const [localSlackAppToken, setLocalSlackAppToken] = useState('');

  useEffect(() => {
    if (settings) {
      setLocalEnabled(settings.enabled);
      setLocalUrl(settings.url);
      const dm =
        settings.defaultChatMode ??
        (settings.default_chat_mode === 'investigation' ? 'investigation' : 'fast');
      setLocalDefaultChatMode(dm === 'investigation' ? 'investigation' : 'fast');
      setLocalFastModel(settings.chatModel ?? settings.chat_model ?? '');
      setLocalInvestigationModel(settings.investigationModel ?? settings.investigation_model ?? '');
      setLocalQanInsightsModel(settings.qanInsightsModel ?? settings.qan_insights_model ?? '');
      setLocalAdreMaxConversationMessages(
        settings.adreMaxConversationMessages ??
          settings.adre_max_conversation_messages ??
          40
      );
      setLocalBehaviorFast(
        hydrateAdreBehaviorMap(behaviorFromSettings(settings, 'behaviorControlsFast', 'behavior_controls_fast'), 'fast')
      );
      setLocalBehaviorInvestigation(
        hydrateAdreBehaviorMap(
          behaviorFromSettings(settings, 'behaviorControlsInvestigation', 'behavior_controls_investigation'),
          'investigation'
        )
      );
      setLocalBehaviorFormat(
        hydrateAdreBehaviorMap(
          behaviorFromSettings(settings, 'behaviorControlsFormatReport', 'behavior_controls_format_report'),
          'format'
        )
      );
      setLocalChatPrompt(settings.chatPromptDisplay ?? settings.chatPrompt ?? '');
      setLocalInvestigationPrompt(settings.investigationPromptDisplay ?? settings.investigationPrompt ?? '');
      setLocalQanInsightsPrompt(
        settings.qanInsightsPromptDisplay ??
          settings.qanInsightsPrompt ??
          settings.qan_insights_prompt_display ??
          settings.qan_insights_prompt ??
          ''
      );
      setLocalServiceNowURL(
        settings.servicenowUrl ?? settings.servicenow_url ?? 'https://perconadev.service-now.com/api/pellc/percona_connector/create'
      );
      setLocalPromptMaxBytes(settings.promptMaxBytes ?? settings.prompt_max_bytes ?? 16 * 1024);
      setLocalSlackEnabled(settings.slackEnabled ?? settings.slack_enabled ?? false);
      setLocalSlackBotToken('');
      setLocalSlackAppToken('');
    }
  }, [settings]);

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
                    Slack integration
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    Optional Socket Mode bot for @mentions and thread replies (runs on the PMM HA leader).
                    {(settings?.slackConfigured ?? settings?.slack_configured) && (
                      <Chip label="Tokens saved" size="small" color="success" sx={{ ml: 1 }} />
                    )}
                  </Typography>
                  <FormControlLabel
                    control={
                      <Switch
                        checked={localSlackEnabled}
                        onChange={(_e: SyntheticEvent, v: boolean) => setLocalSlackEnabled(v)}
                        disabled={!localEnabled || !localUrl.trim()}
                      />
                    }
                    label="Enable Slack bot"
                  />
                  {!localEnabled || !localUrl.trim() ? (
                    <Typography variant="caption" color="text.secondary">
                      Enable ADRE and set HolmesGPT URL first.
                    </Typography>
                  ) : null}
                  <Typography variant="body2" color="text.secondary">
                    Clickable Grafana links in Slack use <strong>Public address</strong> from{' '}
                    <strong>PMM Settings → Advanced</strong> when that field is set.
                  </Typography>
                  <TextField
                    label="Slack Bot User OAuth Token"
                    type="password"
                    placeholder="Leave empty to keep existing value"
                    value={localSlackBotToken}
                    onChange={(e: ChangeEvent<HTMLInputElement>) => setLocalSlackBotToken(e.target.value)}
                    size="small"
                    fullWidth
                    disabled={!localSlackEnabled}
                    helperText="xoxb-… from your Slack app; leave empty to keep current"
                  />
                  <TextField
                    label="Slack App-Level Token (Socket Mode)"
                    type="password"
                    placeholder="Leave empty to keep existing value"
                    value={localSlackAppToken}
                    onChange={(e: ChangeEvent<HTMLInputElement>) => setLocalSlackAppToken(e.target.value)}
                    size="small"
                    fullWidth
                    disabled={!localSlackEnabled}
                    helperText="xapp-… with connections:write; leave empty to keep current"
                  />
                </Stack>
                <Divider />
                <Stack gap={2}>
                  <Typography variant="subtitle1" fontWeight={600}>
                    ADRE panel &amp; Holmes
                  </Typography>
                  <FormControl size="small" fullWidth>
                    <InputLabel>Default mode in ADRE panel</InputLabel>
                    <Select
                      value={localDefaultChatMode}
                      label="Default mode in ADRE panel"
                      onChange={(e: SelectChangeEvent<'fast' | 'investigation'>) =>
                        setLocalDefaultChatMode(e.target.value as 'fast' | 'investigation')
                      }
                    >
                      <MenuItem value="fast">Fast</MenuItem>
                      <MenuItem value="investigation">Investigation</MenuItem>
                    </Select>
                  </FormControl>
                  <FormControl size="small" fullWidth>
                    <InputLabel>Fast mode model</InputLabel>
                    <Select
                      value={localFastModel}
                      label="Fast mode model"
                      onChange={(e: SelectChangeEvent<string>) => setLocalFastModel(e.target.value)}
                    >
                      <MenuItem value="">Holmes default</MenuItem>
                      {models.map((m) => (
                        <MenuItem key={`fast-${m}`} value={m}>
                          {m}
                        </MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                  <FormControl size="small" fullWidth>
                    <InputLabel>Investigation mode model</InputLabel>
                    <Select
                      value={localInvestigationModel}
                      label="Investigation mode model"
                      onChange={(e: SelectChangeEvent<string>) =>
                        setLocalInvestigationModel(e.target.value)
                      }
                    >
                      <MenuItem value="">Holmes default</MenuItem>
                      {models.map((m) => (
                        <MenuItem key={`investigation-${m}`} value={m}>
                          {m}
                        </MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                  <FormControl size="small" fullWidth>
                    <InputLabel>QAN Insights model</InputLabel>
                    <Select
                      value={localQanInsightsModel}
                      label="QAN Insights model"
                      onChange={(e: SelectChangeEvent<string>) =>
                        setLocalQanInsightsModel(e.target.value)
                      }
                    >
                      <MenuItem value="">Holmes default</MenuItem>
                      {models.map((m) => (
                        <MenuItem key={`qan-insights-${m}`} value={m}>
                          {m}
                        </MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                  <TextField
                    label="Max conversation messages to Holmes"
                    type="number"
                    inputProps={{ min: 4, max: 200 }}
                    value={localAdreMaxConversationMessages}
                    onChange={(e: ChangeEvent<HTMLInputElement>) =>
                      setLocalAdreMaxConversationMessages(parseInt(e.target.value, 10) || 40)
                    }
                    size="small"
                    fullWidth
                    helperText="Caps conversation_history size (4–200). Reduces Holmes context-overflow failures."
                  />
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
                </Stack>
                <Divider />
                <AdreBehaviorControlsBlock
                  variant="fast"
                  title="Fast mode — behavior controls"
                  description="Tuning for the Fast path in the ADRE chat panel (runbooks, TodoWrite, etc.)."
                  value={localBehaviorFast}
                  onChange={setLocalBehaviorFast}
                  onJsonError={(msg) => enqueueSnackbar(msg, { variant: 'error' })}
                />
                <Divider />
                <AdreBehaviorControlsBlock
                  variant="investigation"
                  title="Investigation mode — behavior controls"
                  description="Used for investigation chat and investigation runs. Empty preset means Holmes defaults for omitted keys."
                  value={localBehaviorInvestigation}
                  onChange={setLocalBehaviorInvestigation}
                  onJsonError={(msg) => enqueueSnackbar(msg, { variant: 'error' })}
                />
                <Divider />
                <AdreBehaviorControlsBlock
                  variant="format"
                  title="Format investigation report — behavior controls"
                  description="Used when PMM asks Holmes to turn a raw investigation report into structured JSON."
                  value={localBehaviorFormat}
                  onChange={setLocalBehaviorFormat}
                  onJsonError={(msg) => enqueueSnackbar(msg, { variant: 'error' })}
                />
                <Divider />
                <Stack gap={2}>
                  <Typography variant="subtitle1" fontWeight={600}>
                    Prompts
                  </Typography>
                  <TextField
                    label="Fast mode prompt"
                    placeholder="Additional system prompt for Fast mode (Holmes additional_system_prompt)"
                    value={localChatPrompt}
                    onChange={(e: ChangeEvent<HTMLInputElement>) => setLocalChatPrompt(e.target.value)}
                    size="small"
                    fullWidth
                    multiline
                    minRows={3}
                    helperText={`Fast mode (${byteCount(localChatPrompt)} / ${localPromptMaxBytes} bytes)`}
                  />
                  <TextField
                    label="Investigation mode prompt"
                    placeholder="Additional system prompt for Investigation mode"
                    value={localInvestigationPrompt}
                    onChange={(e: ChangeEvent<HTMLInputElement>) => setLocalInvestigationPrompt(e.target.value)}
                    size="small"
                    fullWidth
                    multiline
                    minRows={3}
                    helperText={`Investigation mode (${byteCount(localInvestigationPrompt)} / ${localPromptMaxBytes} bytes)`}
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
                        default_chat_mode: localDefaultChatMode,
                        chat_model: localFastModel || undefined,
                        investigation_model: localInvestigationModel || undefined,
                        qan_insights_model: localQanInsightsModel || undefined,
                        adre_max_conversation_messages: localAdreMaxConversationMessages,
                        behavior_controls_fast: localBehaviorFast,
                        behavior_controls_investigation: localBehaviorInvestigation,
                        behavior_controls_format_report: localBehaviorFormat,
                        chat_prompt: localChatPrompt || undefined,
                        investigation_prompt: localInvestigationPrompt || undefined,
                        qan_insights_prompt: localQanInsightsPrompt || undefined,
                        prompt_max_bytes: localPromptMaxBytes,
                        servicenow_url: localServiceNowURL || undefined,
                        ...(localServiceNowAPIKey ? { servicenow_api_key: localServiceNowAPIKey } : {}),
                        ...(localServiceNowClientToken ? { servicenow_client_token: localServiceNowClientToken } : {}),
                        slack_enabled: localSlackEnabled,
                        ...(localSlackBotToken ? { slack_bot_token: localSlackBotToken } : {}),
                        ...(localSlackAppToken ? { slack_app_token: localSlackAppToken } : {}),
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
