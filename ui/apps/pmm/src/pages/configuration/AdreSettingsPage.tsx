import {
  Alert,
  Box,
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
  Tab,
  Tabs,
  TextField,
  Typography,
} from '@mui/material';
import { FC, useState, useEffect, ChangeEvent, SyntheticEvent } from 'react';
import { Page } from 'components/page';
import { useAdreModels, useAdreSettings, useUpdateAdreSettings } from 'hooks/api/useAdre';
import type { AdreSettings } from 'api/adre';
import { AdreBehaviorControlsBlock } from 'pages/configuration/AdreBehaviorControlsBlock';
import { hydrateAdreBehaviorMap } from 'pages/configuration/AdreBehaviorControlsBlock.utils';
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

const pageProps = {
  title: 'AI Assistant Settings',
  fullWidth: true as const,
  surface: 'paper' as const,
  footer: null,
};

const AdreSettingsPage: FC = () => {
  const { user } = useUser();
  const { enqueueSnackbar } = useSnackbar();
  const { data: settings, isLoading, isError, error } = useAdreSettings();
  // Models come from the Holmes backend, which is only reachable when ADRE is enabled; fetching while
  // disabled returns 400 "ADRE is disabled" and would surface a spurious error toast on this page.
  const adreEnabled = settings?.enabled ?? false;
  const { data: models = [] } = useAdreModels({ enabled: adreEnabled });
  const updateSettings = useUpdateAdreSettings();
  const [tab, setTab] = useState(0);
  const [localEnabled, setLocalEnabled] = useState(settings?.enabled ?? false);
  const [localUrl, setLocalUrl] = useState(settings?.url ?? '');
  const [localTlsSkipVerify, setLocalTlsSkipVerify] = useState(
    settings?.tlsSkipVerify ?? settings?.tls_skip_verify ?? false
  );
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
  const [localSlackAutoInvestigate, setLocalSlackAutoInvestigate] = useState(
    settings?.slackAutoInvestigate ?? settings?.slack_auto_investigate ?? false
  );
  const [localSlackBotToken, setLocalSlackBotToken] = useState('');
  const [localSlackAppToken, setLocalSlackAppToken] = useState('');
  const [localSlackAllowedChannels, setLocalSlackAllowedChannels] = useState(
    (settings?.slackAllowedChannels ?? settings?.slack_allowed_channels ?? []).join('\n')
  );
  const [localSlackAllowedUsers, setLocalSlackAllowedUsers] = useState(
    (settings?.slackAllowedUsers ?? settings?.slack_allowed_users ?? []).join('\n')
  );
  const [localSlackAutoInvestigateChannels, setLocalSlackAutoInvestigateChannels] = useState(
    (settings?.slackAutoInvestigateChannels ?? settings?.slack_auto_investigate_channels ?? []).join('\n')
  );
  const [localSlackAlertBotIds, setLocalSlackAlertBotIds] = useState(
    (settings?.slackAlertBotIds ?? settings?.slack_alert_bot_ids ?? []).join('\n')
  );
  const [localAutoInvestigateMinSeverity, setLocalAutoInvestigateMinSeverity] = useState(
    settings?.autoInvestigateMinSeverity ?? settings?.auto_investigate_min_severity ?? ''
  );
  const [localAutoInvestigateLabelMatchers, setLocalAutoInvestigateLabelMatchers] = useState(
    (settings?.autoInvestigateLabelMatchers ?? settings?.auto_investigate_label_matchers ?? []).join('\n')
  );
  const [localAutoInvestigateHourlyCap, setLocalAutoInvestigateHourlyCap] = useState(
    settings?.autoInvestigateHourlyCap ?? settings?.auto_investigate_hourly_cap ?? 0
  );

  useEffect(() => {
    if (settings) {
      setLocalEnabled(settings.enabled);
      setLocalUrl(settings.url ?? '');
      setLocalTlsSkipVerify(settings.tlsSkipVerify ?? settings.tls_skip_verify ?? false);
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
      setLocalSlackAutoInvestigate(
        settings.slackAutoInvestigate ?? settings.slack_auto_investigate ?? false
      );
      setLocalSlackBotToken('');
      setLocalSlackAppToken('');
      setLocalSlackAllowedChannels(
        (settings.slackAllowedChannels ?? settings.slack_allowed_channels ?? []).join('\n')
      );
      setLocalSlackAllowedUsers(
        (settings.slackAllowedUsers ?? settings.slack_allowed_users ?? []).join('\n')
      );
      setLocalSlackAutoInvestigateChannels(
        (settings.slackAutoInvestigateChannels ?? settings.slack_auto_investigate_channels ?? []).join('\n')
      );
      setLocalSlackAlertBotIds(
        (settings.slackAlertBotIds ?? settings.slack_alert_bot_ids ?? []).join('\n')
      );
      setLocalAutoInvestigateMinSeverity(
        settings.autoInvestigateMinSeverity ?? settings.auto_investigate_min_severity ?? ''
      );
      setLocalAutoInvestigateLabelMatchers(
        (settings.autoInvestigateLabelMatchers ?? settings.auto_investigate_label_matchers ?? []).join('\n')
      );
      setLocalAutoInvestigateHourlyCap(
        settings.autoInvestigateHourlyCap ?? settings.auto_investigate_hourly_cap ?? 0
      );
    }
  }, [settings]);

  const isAdmin = user?.isPMMAdmin ?? false;
  const isForbidden = isError && isForbiddenError(error);

  const splitList = (s: string): string[] =>
    s
      .split(/[\n,]/)
      .map((x) => x.trim())
      .filter(Boolean);

  // Label matchers can contain commas in the value (e.g. team=db,ops), so split on newlines only.
  const splitLines = (s: string): string[] =>
    s
      .split('\n')
      .map((x) => x.trim())
      .filter(Boolean);

  const onSave = () =>
    updateSettings.mutate(
      {
        enabled: localEnabled,
        url: localUrl,
        tls_skip_verify: localTlsSkipVerify,
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
        slack_auto_investigate: localSlackAutoInvestigate,
        ...(localSlackBotToken ? { slack_bot_token: localSlackBotToken } : {}),
        ...(localSlackAppToken ? { slack_app_token: localSlackAppToken } : {}),
        slack_allowed_channels: splitList(localSlackAllowedChannels),
        slack_allowed_users: splitList(localSlackAllowedUsers),
        slack_auto_investigate_channels: splitList(localSlackAutoInvestigateChannels),
        slack_alert_bot_ids: splitList(localSlackAlertBotIds),
        auto_investigate_min_severity: localAutoInvestigateMinSeverity,
        auto_investigate_label_matchers: splitLines(localAutoInvestigateLabelMatchers),
        auto_investigate_hourly_cap: Math.max(0, Number(localAutoInvestigateHourlyCap) || 0),
      } as Partial<AdreSettings> & Record<string, unknown>,
      {
        onError: (err: unknown) => {
          const msg =
            (err as { response?: { data?: { error?: string } } })?.response?.data?.error ??
            (err as Error)?.message ??
            'Failed to save settings';
          enqueueSnackbar(msg, { variant: 'error' });
        },
        onSuccess: () => {
          enqueueSnackbar('Settings saved', { variant: 'success' });
        },
      }
    );

  if (isLoading) {
    return (
      <Page {...pageProps}>
        <Typography>Loading...</Typography>
      </Page>
    );
  }

  if (isError && !isForbidden) {
    return (
      <Page {...pageProps}>
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
      <Page {...pageProps}>
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

  if (!isAdmin) {
    return (
      <Page {...pageProps}>
        <Card variant="outlined" sx={{ maxWidth: 720, mx: 'auto', width: '100%' }}>
          <CardContent>
            <Alert severity="info">
              Admin access is required to modify AI Assistant settings. Contact your administrator or
              open PMM Settings.
            </Alert>
          </CardContent>
        </Card>
      </Page>
    );
  }

  return (
    <Page {...pageProps}>
      <Box
        sx={{ flex: 1, minHeight: 0, minWidth: 0, overflowY: 'auto', WebkitOverflowScrolling: 'touch', p: 2 }}
        data-testid="adre-settings-scroll"
      >
        <Box sx={{ maxWidth: 720, mx: 'auto', width: '100%' }}>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
            Configure the Autonomous Database Reliability Engineer (ADRE) and AI backend for
            AI-assisted investigations.
          </Typography>

          <Tabs
            value={tab}
            onChange={(_: SyntheticEvent, v: number) => setTab(v)}
            variant="scrollable"
            scrollButtons="auto"
            sx={{ mb: 2 }}
          >
            <Tab label="General" />
            <Tab label="Prompts" />
            <Tab label="Behavior" />
            <Tab label="Slack" />
            <Tab label="ServiceNow" />
          </Tabs>

          {tab === 0 && (
            <Card variant="outlined">
              <CardContent>
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
                      label="AI service URL"
                      placeholder="http://localhost:8080"
                      value={localUrl}
                      onChange={(e: ChangeEvent<HTMLInputElement>) => setLocalUrl(e.target.value)}
                      size="small"
                      fullWidth
                    />
                    <FormControlLabel
                      control={
                        <Switch
                          checked={localTlsSkipVerify}
                          onChange={(_e: SyntheticEvent, v: boolean) => setLocalTlsSkipVerify(v)}
                          disabled={!localEnabled || !(localUrl ?? '').trim().startsWith('https://')}
                        />
                      }
                      label="Skip TLS certificate verification"
                    />
                    {localTlsSkipVerify ? (
                      <Alert severity="warning" sx={{ py: 0 }}>
                        PMM will not verify the Holmes TLS certificate. Use only for development or
                        trusted networks with self-signed certificates.
                      </Alert>
                    ) : (
                      <Typography variant="caption" color="text.secondary" display="block">
                        Available when the AI service URL uses https. You can also set{' '}
                        <strong>PMM_ADRE_TLS_SKIP_VERIFY=true</strong> at container startup.
                      </Typography>
                    )}
                  </Stack>
                  <Divider />
                  <Stack gap={2}>
                    <Typography variant="subtitle1" fontWeight={600}>
                      ADRE panel
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
                        <MenuItem value="">Service default</MenuItem>
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
                        <MenuItem value="">Service default</MenuItem>
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
                        <MenuItem value="">Service default</MenuItem>
                        {models.map((m) => (
                          <MenuItem key={`qan-insights-${m}`} value={m}>
                            {m}
                          </MenuItem>
                        ))}
                      </Select>
                    </FormControl>
                    <TextField
                      label="Max conversation messages to AI"
                      type="number"
                      inputProps={{ min: 4, max: 200 }}
                      value={localAdreMaxConversationMessages}
                      onChange={(e: ChangeEvent<HTMLInputElement>) =>
                        setLocalAdreMaxConversationMessages(parseInt(e.target.value, 10) || 40)
                      }
                      size="small"
                      fullWidth
                      helperText="Caps conversation_history size (4–200). Reduces context-overflow failures."
                    />
                    <TextField
                      label="Prompt max bytes"
                      type="number"
                      inputProps={{ min: 1024, max: 65536 }}
                      value={localPromptMaxBytes}
                      onChange={(e: ChangeEvent<HTMLInputElement>) =>
                        setLocalPromptMaxBytes(parseInt(e.target.value, 10) || 16 * 1024)
                      }
                      size="small"
                      fullWidth
                      helperText="Allowed range: 1024–65536. Default recommended: 16384."
                    />
                  </Stack>
                </Stack>
              </CardContent>
            </Card>
          )}

          {tab === 1 && (
            <Card variant="outlined">
              <CardContent>
                <Stack gap={2}>
                  <Typography variant="subtitle1" fontWeight={600}>
                    Prompts
                  </Typography>
                  <TextField
                    label="Fast mode prompt"
                    placeholder="Additional system prompt for Fast mode"
                    value={localChatPrompt}
                    onChange={(e: ChangeEvent<HTMLInputElement>) => setLocalChatPrompt(e.target.value)}
                    size="small"
                    fullWidth
                    multiline
                    minRows={4}
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
                    minRows={4}
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
                    minRows={4}
                    helperText={`Used when analyzing a query from Query Analytics; leave empty for default (${byteCount(localQanInsightsPrompt)} / ${localPromptMaxBytes} bytes)`}
                  />
                </Stack>
              </CardContent>
            </Card>
          )}

          {tab === 2 && (
            <Stack gap={2}>
              <AdreBehaviorControlsBlock
                variant="fast"
                title="Fast mode — behavior controls"
                description="Tuning for the Fast path in the ADRE chat panel (tools / TodoWrite, etc.)."
                value={localBehaviorFast}
                onChange={setLocalBehaviorFast}
                onJsonError={(msg) => enqueueSnackbar(msg, { variant: 'error' })}
              />
              <AdreBehaviorControlsBlock
                variant="investigation"
                title="Investigation mode — behavior controls"
                description="Used for investigation chat and investigation runs. Empty preset means service defaults for omitted keys."
                value={localBehaviorInvestigation}
                onChange={setLocalBehaviorInvestigation}
                onJsonError={(msg) => enqueueSnackbar(msg, { variant: 'error' })}
              />
              <AdreBehaviorControlsBlock
                variant="format"
                title="Format investigation report — behavior controls"
                description="Used when PMM turns a raw investigation report into structured JSON."
                value={localBehaviorFormat}
                onChange={setLocalBehaviorFormat}
                onJsonError={(msg) => enqueueSnackbar(msg, { variant: 'error' })}
              />
            </Stack>
          )}

          {tab === 3 && (
            <Card variant="outlined">
              <CardContent>
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
                        onChange={(_e: SyntheticEvent, v: boolean) => {
                          setLocalSlackEnabled(v);
                          if (!v) {
                            setLocalSlackAutoInvestigate(false);
                          }
                        }}
                        disabled={!localEnabled || !localUrl.trim()}
                      />
                    }
                    label="Enable Slack bot"
                  />
                  <FormControlLabel
                    control={
                      <Switch
                        checked={localSlackAutoInvestigate}
                        onChange={(_e: SyntheticEvent, v: boolean) => setLocalSlackAutoInvestigate(v)}
                        disabled={!localSlackEnabled}
                      />
                    }
                    label="Auto-investigate firing alerts (Grafana Alertmanager)"
                  />
                  <Typography variant="caption" color="text.secondary" display="block">
                    When enabled, firing alerts from Grafana Alertmanager (via a reconciliation poll, plus
                    an optional auto-provisioned webhook) create one investigation per alert episode and post
                    a summary to the output channels below. Bound the cost with the severity floor / label
                    matchers / hourly cap.
                  </Typography>
                  {!localEnabled || !localUrl.trim() ? (
                    <Typography variant="caption" color="text.secondary">
                      Enable ADRE and set the AI service URL first (General tab).
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
                  <Divider sx={{ my: 1 }} />
                  <Typography variant="subtitle2" fontWeight={600}>
                    Human chat allowlists (fail-closed)
                  </Typography>
                  <Typography variant="caption" color="text.secondary" display="block">
                    The bot replies only in listed channels to listed users. Leave a list empty and it
                    answers no one. Use Slack object IDs (channels Cxxxx, users Uxxxx), one per line.
                  </Typography>
                  <TextField
                    label="Allowed channels"
                    value={localSlackAllowedChannels}
                    onChange={(e: ChangeEvent<HTMLInputElement>) => setLocalSlackAllowedChannels(e.target.value)}
                    size="small"
                    fullWidth
                    multiline
                    minRows={2}
                    disabled={!localSlackEnabled}
                    placeholder={'C0123ABCD\nC0456EFGH'}
                  />
                  <TextField
                    label="Allowed users"
                    value={localSlackAllowedUsers}
                    onChange={(e: ChangeEvent<HTMLInputElement>) => setLocalSlackAllowedUsers(e.target.value)}
                    size="small"
                    fullWidth
                    multiline
                    minRows={2}
                    disabled={!localSlackEnabled}
                    placeholder={'U0123ABCD\nU0456EFGH'}
                  />
                  <Divider sx={{ my: 1 }} />
                  <Typography variant="subtitle2" fontWeight={600}>
                    Auto-investigate output &amp; cost guards
                  </Typography>
                  <TextField
                    label="Alert channels"
                    value={localSlackAutoInvestigateChannels}
                    onChange={(e: ChangeEvent<HTMLInputElement>) =>
                      setLocalSlackAutoInvestigateChannels(e.target.value)
                    }
                    size="small"
                    fullWidth
                    multiline
                    minRows={2}
                    disabled={!localSlackEnabled}
                    placeholder="C0123ABCD"
                    helperText="Channels the bot scrapes for Grafana alert messages and posts the investigation thread into."
                  />
                  <TextField
                    label="Alert bot IDs (optional)"
                    value={localSlackAlertBotIds}
                    onChange={(e: ChangeEvent<HTMLInputElement>) =>
                      setLocalSlackAlertBotIds(e.target.value)
                    }
                    size="small"
                    fullWidth
                    multiline
                    minRows={2}
                    disabled={!localSlackEnabled}
                    placeholder="B0123GRAFANA"
                    helperText="Restrict which Slack bot/app IDs the scrape accepts alerts from (e.g. the Grafana app). Empty ⇒ accept any bot in the alert channels."
                  />
                  <TextField
                    select
                    label="Minimum severity"
                    value={localAutoInvestigateMinSeverity}
                    onChange={(e: ChangeEvent<HTMLInputElement>) =>
                      setLocalAutoInvestigateMinSeverity(e.target.value)
                    }
                    size="small"
                    fullWidth
                    disabled={!localSlackEnabled}
                    helperText="Only auto-investigate alerts at or above this severity."
                  >
                    <MenuItem value="">No floor (all severities)</MenuItem>
                    <MenuItem value="info">info</MenuItem>
                    <MenuItem value="warning">warning</MenuItem>
                    <MenuItem value="critical">critical</MenuItem>
                  </TextField>
                  <TextField
                    label="Label matchers"
                    value={localAutoInvestigateLabelMatchers}
                    onChange={(e: ChangeEvent<HTMLInputElement>) =>
                      setLocalAutoInvestigateLabelMatchers(e.target.value)
                    }
                    size="small"
                    fullWidth
                    multiline
                    minRows={2}
                    disabled={!localSlackEnabled}
                    placeholder={'service_type=mysql\nteam=db'}
                    helperText="Optional key=value matchers (all must match), one per line."
                  />
                  <TextField
                    label="Hourly cap"
                    type="number"
                    value={localAutoInvestigateHourlyCap}
                    onChange={(e: ChangeEvent<HTMLInputElement>) =>
                      setLocalAutoInvestigateHourlyCap(Math.max(0, Math.floor(Number(e.target.value) || 0)))
                    }
                    size="small"
                    fullWidth
                    disabled={!localSlackEnabled}
                    slotProps={{ htmlInput: { min: 0, step: 1 } }}
                    helperText="Max auto-investigations per hour (0 = unbounded)."
                  />
                </Stack>
              </CardContent>
            </Card>
          )}

          {tab === 4 && (
            <Card variant="outlined">
              <CardContent>
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
              </CardContent>
            </Card>
          )}

          <Box sx={{ mt: 2 }}>
            <Button variant="contained" onClick={onSave} disabled={updateSettings.isPending}>
              {updateSettings.isPending ? 'Saving...' : 'Save'}
            </Button>
            <Typography variant="caption" color="text.secondary" sx={{ ml: 2 }}>
              Save applies changes from all tabs.
            </Typography>
          </Box>
        </Box>
      </Box>
    </Page>
  );
};

export default AdreSettingsPage;
