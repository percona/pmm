import { api } from './api';

/** Holmes behavior_controls keys supported by PMM (see Holmes HTTP API — fast mode / prompt controls). */
export const ADRE_BEHAVIOR_CONTROL_KEYS = [
  'intro',
  'ask_user',
  'todowrite_instructions',
  'todowrite_reminder',
  'ai_safety',
  'toolset_instructions',
  'permission_errors',
  'general_instructions',
  'style_guide',
  'cluster_name',
  'system_prompt_additions',
  'files',
  'time_skills',
] as const;

export type AdreBehaviorControlsMap = Partial<Record<(typeof ADRE_BEHAVIOR_CONTROL_KEYS)[number], boolean>>;

export interface AdreSettings {
  enabled: boolean;
  url: string;
  chatPrompt?: string;
  investigationPrompt?: string;
  /** Default Holmes model alias for Fast mode chat. Empty uses Holmes default. */
  chatModel?: string;
  chat_model?: string;
  /** Default Holmes model alias for Investigation mode chat. Empty uses Holmes default. */
  investigationModel?: string;
  investigation_model?: string;
  /** Display value when chat_prompt is empty (built-in default). */
  chatPromptDisplay?: string;
  /** Display value when investigation_prompt is empty (built-in default). */
  investigationPromptDisplay?: string;
  /** Default ADRE panel mode when the UI does not override. */
  defaultChatMode?: 'fast' | 'investigation' | 'chat';
  default_chat_mode?: string;
  /** Holmes behavior_controls for Fast mode. Empty {} uses PMM shipped preset when calling Holmes. */
  behaviorControlsFast?: AdreBehaviorControlsMap;
  behavior_controls_fast?: Record<string, boolean>;
  behaviorControlsInvestigation?: AdreBehaviorControlsMap;
  behavior_controls_investigation?: Record<string, boolean>;
  behaviorControlsFormatReport?: AdreBehaviorControlsMap;
  behavior_controls_format_report?: Record<string, boolean>;
  /** Max messages in conversation_history sent to Holmes (4–200; 0 = server default). */
  adreMaxConversationMessages?: number;
  adre_max_conversation_messages?: number;
  /** System prompt for QAN AI Insights. Empty = use built-in default. */
  qanInsightsPrompt?: string;
  /** Default Holmes model alias for QAN AI Insights. Empty uses Holmes default. */
  qanInsightsModel?: string;
  /** Display value when qan_insights_prompt is empty (built-in default). */
  qanInsightsPromptDisplay?: string;
  qan_insights_prompt?: string;
  qan_insights_prompt_display?: string;
  qan_insights_model?: string;
  /** ServiceNow Percona Connector API URL. */
  servicenowUrl?: string;
  servicenow_url?: string;
  /** ServiceNow API key (x-sn-apikey header). Only sent when saving; backend never exposes the raw value on GET. */
  servicenowApiKey?: string;
  servicenow_api_key?: string;
  /** ServiceNow client token. Only sent when saving. */
  servicenowClientToken?: string;
  servicenow_client_token?: string;
  /** True when URL + API key + client token are all configured server-side. */
  servicenowConfigured?: boolean;
  servicenow_configured?: boolean;
  /** Max bytes allowed for ADRE prompts. */
  promptMaxBytes?: number;
  prompt_max_bytes?: number;
  /** PMM-managed Slack bot (Socket Mode). */
  slackEnabled?: boolean;
  slack_enabled?: boolean;
  /** Auto-run ADRE on bot messages containing FIRING (Slack alerts). */
  slackAutoInvestigate?: boolean;
  slack_auto_investigate?: boolean;
  /** True when bot + app tokens are stored server-side (GET never returns raw tokens). */
  slackConfigured?: boolean;
  slack_configured?: boolean;
  slackBotToken?: string;
  slack_bot_token?: string;
  slackAppToken?: string;
  slack_app_token?: string;
  /** Skip TLS certificate verification for PMM → HolmesGPT. */
  tlsSkipVerify?: boolean;
  tls_skip_verify?: boolean;
  /** Slack human-chat allowlists (fail-closed) — Slack object IDs. */
  slackAllowedChannels?: string[];
  slack_allowed_channels?: string[];
  slackAllowedUsers?: string[];
  slack_allowed_users?: string[];
  /** Alert channels: scraped for Grafana alert messages and where the investigation thread is posted. */
  slackAutoInvestigateChannels?: string[];
  slack_auto_investigate_channels?: string[];
  /** Optional allow-list of Slack bot/app IDs the scrape accepts alerts from (empty ⇒ any bot). */
  slackAlertBotIds?: string[];
  slack_alert_bot_ids?: string[];
  /** Auto-investigate selection + cost guards. */
  autoInvestigateMinSeverity?: string;
  auto_investigate_min_severity?: string;
  autoInvestigateLabelMatchers?: string[];
  auto_investigate_label_matchers?: string[];
  autoInvestigateHourlyCap?: number;
  auto_investigate_hourly_cap?: number;
}

export interface AdreModelsResponse {
  modelName: string[];
}

export interface AdreChatRequest {
  ask: string;
  /** Required by pmm-managed; server loads trimmed history from PostgreSQL. */
  conversation_id?: number;
  conversation_history?: unknown[];
  model?: string;
  stream?: boolean;
  /** Server resolves prompt and behavior_controls from mode; client must not send additionalSystemPrompt. */
  mode?: 'fast' | 'investigation' | 'chat';
  pageContext?: unknown;
  /** Structured Grafana context; pmm-managed merges into Holmes additional_system_prompt. */
  dashboard_context?: string;
  frontend_tools?: unknown[];
  frontend_tool_results?: unknown[];
  tool_decisions?: unknown[];
}

export interface AdreQanInsightsRequest {
  serviceId: string;
  queryText: string;
  queryId?: string;
  fingerprint?: string;
  timeFrom?: string;
  timeTo?: string;
  force?: boolean;
}

export interface AdreQanInsightsResponse {
  analysis: string;
  created_at?: string;
  createdAt?: string;
  cached?: boolean;
  usage?: HolmesUsage;
}

export interface AdreUsageTotals {
  totalTokens: number;
  total_tokens?: number;
  cachedTokens: number;
  cached_tokens?: number;
  totalCost: number;
  total_cost?: number;
  callCount: number;
  call_count?: number;
}

export interface AdreUsageBucket {
  bucket?: string;
  feature?: string;
  model?: string;
  totalTokens: number;
  total_tokens?: number;
  cachedTokens: number;
  cached_tokens?: number;
  totalCost: number;
  total_cost?: number;
  callCount: number;
  call_count?: number;
}

export interface AdreUsageSummaryResponse {
  from: string;
  to: string;
  totals: AdreUsageTotals;
  series: AdreUsageBucket[];
  byFeature: AdreUsageBucket[];
  by_feature?: AdreUsageBucket[];
  byModel: AdreUsageBucket[];
  by_model?: AdreUsageBucket[];
}

export interface AdreUsageEvent {
  id: number;
  createdAt: string;
  created_at?: string;
  feature: string;
  featureRef: string;
  feature_ref?: string;
  model: string;
  totalTokens?: number;
  total_tokens?: number;
  cachedTokens?: number;
  cached_tokens?: number;
  totalCost?: number;
  total_cost?: number;
  triggeredBy?: string;
  triggered_by?: string;
  stream: boolean;
}

export const getAdreUsageSummary = async (params?: {
  from?: string;
  to?: string;
  groupBy?: string;
  feature?: string;
  model?: string;
}): Promise<AdreUsageSummaryResponse> => {
  const res = await api.get<AdreUsageSummaryResponse>('/adre/usage/summary', {
    params: {
      from: params?.from,
      to: params?.to,
      feature: params?.feature,
      model: params?.model,
      group_by: params?.groupBy ?? 'day',
    },
  });
  return res.data;
};

export const getAdreUsageEvents = async (params?: {
  from?: string;
  to?: string;
  limit?: number;
  offset?: number;
  feature?: string;
  model?: string;
  investigationId?: string;
  format?: string;
}): Promise<{ events: AdreUsageEvent[] }> => {
  const res = await api.get<{ events: AdreUsageEvent[] }>('/adre/usage/events', { params });
  return res.data;
};

export function normalizeHolmesUsage(row: Partial<HolmesUsage & AdreMessageRow>): HolmesUsage {
  return {
    model: row.model,
    promptTokens: row.promptTokens ?? row.prompt_tokens,
    completionTokens: row.completionTokens ?? row.completion_tokens,
    totalTokens: row.totalTokens ?? row.total_tokens,
    cachedTokens: row.cachedTokens ?? row.cached_tokens,
    totalCost: row.totalCost ?? row.total_cost,
  };
}

export const getAdreSettings = async (): Promise<AdreSettings> => {
  const res = await api.get<AdreSettings>('/adre/settings');
  return res.data;
};

export const updateAdreSettings = async (
  body: Partial<AdreSettings>
): Promise<AdreSettings> => {
  const res = await api.post<AdreSettings>('/adre/settings', body);
  return res.data;
};

export const getAdreModels = async (): Promise<string[]> => {
  const res = await api.get<AdreModelsResponse>('/adre/models');
  return res.data.modelName || [];
};

export const adreQanInsights = async (
  body: AdreQanInsightsRequest
): Promise<AdreQanInsightsResponse> => {
  const res = await api.post<AdreQanInsightsResponse>('/adre/qan-insights', body);
  return res.data;
};

export const getQanInsightsCache = async (
  queryId: string,
  serviceId: string
): Promise<AdreQanInsightsResponse | null> => {
  try {
    const res = await api.get<AdreQanInsightsResponse>('/adre/qan-insights', {
      params: { query_id: queryId, service_id: serviceId },
    });
    return res.data;
  } catch {
    return null;
  }
};

/** Callback for adreChatStream: receives content chunks and/or reasoning chunks. */
export type AdreChatStreamCallback = (content?: string, reasoning?: string) => void;

/** Progress event when HolmesGPT starts or finishes a tool call (SSE events start_tool_calling, tool_calling_result). */
export interface AdreStreamProgressEvent {
  type: 'start_tool' | 'tool_result';
  id: string;
  toolName: string;
  description?: string;
  /** Present when type is 'tool_result'. */
  result?: { status?: string; error?: string | null; data?: unknown };
}

export interface AdreChatStreamOptions {
  onChunk: AdreChatStreamCallback;
  onProgress?: (event: AdreStreamProgressEvent) => void;
  onFrontendToolsRequired?: (payload: {
    pending_frontend_tool_calls: Array<{
      tool_call_id: string;
      tool_name: string;
      arguments?: Record<string, unknown>;
    }>;
    conversation_history: unknown[];
  }) => Promise<Array<{ tool_call_id: string; tool_name: string; result: string }>>;
}

export const adreChatStream = async (
  body: AdreChatRequest,
  onChunkOrOptions: AdreChatStreamCallback | AdreChatStreamOptions
): Promise<void> => {
  const onChunk: AdreChatStreamCallback =
    typeof onChunkOrOptions === 'function'
      ? onChunkOrOptions
      : onChunkOrOptions.onChunk;
  const onProgress = typeof onChunkOrOptions === 'function' ? undefined : onChunkOrOptions.onProgress;
  const onFrontendToolsRequired =
    typeof onChunkOrOptions === 'function' ? undefined : onChunkOrOptions.onFrontendToolsRequired;

  const response = await fetch('/v1/adre/chat', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify({ ...body, stream: true }),
  });
  if (!response.ok) {
    const err = await response.json().catch(() => ({}));
    throw new Error(err.error || `Chat failed: ${response.status}`);
  }
  const reader = response.body?.getReader();
  if (!reader) throw new Error('No response body');
  const decoder = new TextDecoder();
  let buffer = '';
  let lastEvent = '';
  for (;;) {
    const { done, value } = await reader.read();
    if (done) break;
    buffer += decoder.decode(value, { stream: true });
    const lines = buffer.split('\n');
    buffer = lines.pop() || '';
    for (const line of lines) {
      if (line.startsWith('event: ')) {
        lastEvent = line.slice(7).trim();
        continue;
      }
      if (!line.startsWith('data: ')) continue;
      const data = line.slice(6);
      if (!data.trim() || data.trim() === '[DONE]') continue;
      const trimmed = data.trim();
      if (lastEvent === 'start_tool_calling' && trimmed.startsWith('{')) {
        try {
          const o = JSON.parse(trimmed) as Record<string, unknown>;
          const id = typeof o.id === 'string' ? o.id : '';
          const toolName = typeof o.tool_name === 'string' ? o.tool_name : '';
          onProgress?.({
            type: 'start_tool',
            id,
            toolName,
            description: typeof o.description === 'string' ? o.description : undefined,
          });
        } catch {
          // ignore parse errors
        }
        continue;
      }
      if (lastEvent === 'tool_calling_result' && trimmed.startsWith('{')) {
        try {
          const o = JSON.parse(trimmed) as Record<string, unknown>;
          const toolCallId = typeof o.tool_call_id === 'string' ? o.tool_call_id : '';
          const name = typeof o.name === 'string' ? o.name : '';
          const result = o.result && typeof o.result === 'object' ? (o.result as AdreStreamProgressEvent['result']) : undefined;
          onProgress?.({
            type: 'tool_result',
            id: toolCallId,
            toolName: name,
            description: typeof o.description === 'string' ? o.description : undefined,
            result,
          });
        } catch {
          // ignore parse errors
        }
        continue;
      }
      // Holmes stream_chat_formatter: event "error" (e.g. rate limit) with JSON { description, msg, error_code, success }.
      // parseSSEData does not read msg/description, so without this branch the UI would appear to stop with no message.
      if (lastEvent === 'error') {
        const text = formatHolmesStreamError(trimmed);
        throw new Error(text);
      }
      if (lastEvent === 'approval_required' && trimmed.startsWith('{')) {
        try {
          const o = JSON.parse(trimmed) as {
            pending_approvals?: Array<{
              tool_call_id: string;
              tool_name: string;
            }>;
            pending_frontend_tool_calls?: Array<{
              tool_call_id: string;
              tool_name: string;
              arguments?: Record<string, unknown>;
            }>;
            conversation_history?: unknown[];
          };
          if (onFrontendToolsRequired && (o.pending_frontend_tool_calls?.length ?? 0) > 0) {
            const results = await onFrontendToolsRequired({
              pending_frontend_tool_calls: o.pending_frontend_tool_calls ?? [],
              conversation_history: o.conversation_history ?? [],
            });
            await adreChatStream(
              {
                ...body,
                stream: true,
                conversation_history: o.conversation_history ?? [],
                frontend_tool_results: results,
              },
              onChunkOrOptions
            );
            return;
          }
          if ((o.pending_approvals?.length ?? 0) > 0) {
            const names = (o.pending_approvals ?? [])
              .map((a) => a.tool_name)
              .filter(Boolean)
              .join(', ');
            throw new Error(
              `Approval required for backend tool(s): ${names || 'unknown'}. Interactive approval flow is not supported in PMM chat stream yet.`
            );
          }
        } catch (e) {
          if (e instanceof Error) throw e;
          // ignore parse errors
        }
      }
      const parsed = parseSSEData(trimmed);
      if (parsed.content) onChunk(parsed.content);
      if (parsed.reasoning) onChunk(undefined, parsed.reasoning);
    }
  }
};

/** API JSON uses snake_case; axios-case-converter exposes camelCase on the client. */
export interface AdreConversation {
  id: number;
  title: string;
  createdAt: string;
  updatedAt: string;
  lastMessageAt: string;
}

export interface HolmesUsage {
  model?: string;
  promptTokens?: number;
  prompt_tokens?: number;
  completionTokens?: number;
  completion_tokens?: number;
  totalTokens?: number;
  total_tokens?: number;
  cachedTokens?: number;
  cached_tokens?: number;
  totalCost?: number;
  total_cost?: number;
}

export interface AdreMessageRow {
  id: number;
  conversationId: number;
  role: string;
  content: string;
  createdAt: string;
  model?: string;
  toolName?: string;
  toolResultJson?: unknown;
  promptTokens?: number;
  prompt_tokens?: number;
  completionTokens?: number;
  completion_tokens?: number;
  totalTokens?: number;
  total_tokens?: number;
  cachedTokens?: number;
  cached_tokens?: number;
  totalCost?: number;
  total_cost?: number;
}

export interface AdreSearchHit {
  messageId: number;
  conversationId: number;
  role: string;
  snippet: string;
  createdAt: string;
}

export const listAdreConversations = async (params?: {
  limit?: number;
  cursor?: string;
  q?: string;
}): Promise<{ conversations: AdreConversation[]; nextCursor?: string }> => {
  const res = await api.get<{ conversations: AdreConversation[]; nextCursor?: string }>('/adre/conversations', {
    params,
  });
  return res.data;
};

export const createAdreConversation = async (body?: { title?: string }): Promise<AdreConversation> => {
  const res = await api.post<AdreConversation>('/adre/conversations', body ?? {});
  return res.data;
};

export const patchAdreConversation = async (
  id: number,
  body: { title: string }
): Promise<{ id: number; title: string; updated_at: string }> => {
  const res = await api.patch(`/adre/conversations/${id}`, body);
  return res.data;
};

export const deleteAdreConversation = async (id: number): Promise<void> => {
  await api.delete(`/adre/conversations/${id}`);
};

export const getAdreMessages = async (
  conversationId: number,
  params?: { limit?: number; before?: number; after?: number }
): Promise<{ messages: AdreMessageRow[] }> => {
  const res = await api.get<{ messages: AdreMessageRow[] }>(`/adre/conversations/${conversationId}/messages`, {
    params,
  });
  return res.data;
};

export const searchAdreMessages = async (
  q: string,
  limit?: number
): Promise<{ hits: AdreSearchHit[] }> => {
  const res = await api.get<{ hits: AdreSearchHit[] }>('/adre/messages/search', {
    params: { q, limit },
  });
  return res.data;
};

export const getAdreAlerts = async (): Promise<unknown> => {
  const res = await api.get('/adre/alerts');
  return res.data;
};

export interface AlertMetadataFromLabels {
  nodeName?: string;
  serviceName?: string;
  clusterName?: string;
  severity?: string;
}

/** Extract node/service/cluster/severity from alert labels (PMM/VictoriaMetrics conventions). Supports both camelCase and snake_case (axios-case-converter). */
export function getAlertMetadataFromLabels(
  labels?: Record<string, string>
): AlertMetadataFromLabels {
  if (!labels) return {};
  const instanceRaw = labels.instance;
  const nodeFromInstance =
    instanceRaw != null && instanceRaw.includes(':')
      ? instanceRaw.split(':')[0]
      : instanceRaw;
  return {
    nodeName:
      labels.node ??
      labels.nodeName ??
      labels.node_name ??
      labels.nodename ??
      nodeFromInstance ??
      undefined,
    serviceName:
      labels.serviceName ??
      labels.service_name ??
      labels.service ??
      labels.job ??
      undefined,
    clusterName:
      labels.clusterName ?? labels.cluster ?? labels.cluster_name ?? undefined,
    severity: labels.severity ?? labels.Severity ?? undefined,
  };
}

/** Human-readable text from Holmes SSE error payload (event: error). */
function formatHolmesStreamError(data: string): string {
  const trimmed = data.trim();
  if (!trimmed) return 'Request failed';
  if (trimmed.startsWith('{')) {
    try {
      const o = JSON.parse(trimmed) as Record<string, unknown>;
      const msg = typeof o.msg === 'string' ? o.msg.trim() : '';
      const desc = typeof o.description === 'string' ? o.description.trim() : '';
      const code = o.error_code != null ? String(o.error_code) : '';
      const parts = [msg, desc].filter((p) => p.length > 0);
      let out = parts.length > 0 ? parts.join(' — ') : 'Request failed';
      if (code && !out.includes(code)) out = `${out} (code ${code})`;
      return out.length > 6000 ? `${out.slice(0, 6000)}…` : out;
    } catch {
      return trimmed.length > 6000 ? `${trimmed.slice(0, 6000)}…` : trimmed;
    }
  }
  return trimmed.length > 6000 ? `${trimmed.slice(0, 6000)}…` : trimmed;
}

/** Parses SSE data; returns content and/or reasoning from common Holmes/LLM stream fields. */
function parseSSEData(data: string): { content?: string; reasoning?: string } {
  const trimmed = data.trim();
  if (!trimmed || trimmed === '[DONE]') return {};
  if (trimmed.startsWith('{')) {
    try {
      const o = JSON.parse(trimmed) as Record<string, unknown>;
      const contentRaw =
        o.text ?? o.delta ?? o.content ?? o.analysis ?? flattenInstructions(o.instructions) ?? flattenSections(o.sections);
      const content = stringFromValue(contentRaw);
      const reasoningRaw = o.reasoning ?? o.thinking ?? o.thought;
      const reasoning = stringFromValue(reasoningRaw);
      return { ...(content && { content }), ...(reasoning && { reasoning }) };
    } catch {
      // not JSON or invalid
    }
  }
  return { content: trimmed };
}

function flattenInstructions(v: unknown): string | undefined {
  if (!Array.isArray(v) || v.length === 0) return undefined;
  const parts = v.map((item) => {
    if (typeof item === 'string') return item;
    if (item && typeof item === 'object') {
      const o = item as Record<string, unknown>;
      if (typeof o.content === 'string') return o.content;
      if (typeof o.text === 'string') return o.text;
    }
    return undefined;
  });
  const filtered = parts.filter((p): p is string => p != null && p !== '');
  return filtered.length > 0 ? filtered.join('\n') : undefined;
}

function flattenSections(v: unknown): string | undefined {
  if (!v || typeof v !== 'object' || Array.isArray(v)) return undefined;
  const sections = v as Record<string, unknown>;
  const parts: string[] = [];
  for (const [key, val] of Object.entries(sections)) {
    if (typeof val === 'string' && val) parts.push(`## ${key}\n${val}`);
  }
  return parts.length > 0 ? parts.join('\n\n') : undefined;
}

function stringFromValue(v: unknown): string | undefined {
  if (typeof v === 'string') return v;
  if (v && typeof v === 'object' && 'content' in v && typeof (v as { content: unknown }).content === 'string') {
    return (v as { content: string }).content;
  }
  return undefined;
}

// --- ADRE deployment config (admin-only): config.yaml, model_list.yaml, skills, provisioning ---

export interface AdreDeploymentModel {
  name: string;
  litellmModel: string;
  apiBase: string;
  /** Whether an api_key is stored (the key itself is write-only and never returned). */
  keyConfigured: boolean;
  /** Optional extra LiteLLM params (YAML) merged into this model's model_list.yaml entry. */
  extraParams: string;
}

export interface AdreDeploymentSkill {
  name: string;
  description: string;
  body: string;
  /** 'builtin' (shipped) or 'user'. */
  source: string;
  enabled: boolean;
}

export interface AdreDeploymentProvisioning {
  pmmUrl: string;
  tokenConfigured: boolean;
  holmesApiKeyConfigured: boolean;
  restartRequired: boolean;
  lastRenderAt?: string;
  renderStatus: string;
  configDir: string;
}

export interface AdreDeployment {
  configYaml: string;
  models: AdreDeploymentModel[];
  skills: AdreDeploymentSkill[];
  provisioning: AdreDeploymentProvisioning;
}

/** Model upsert payload; apiKey empty = keep existing key. */
export interface AdreDeploymentModelInput {
  name: string;
  litellmModel: string;
  apiBase?: string;
  apiKey?: string;
  extraParams?: string;
}

export interface AdreDeploymentSkillInput {
  name: string;
  description?: string;
  body: string;
  enabled?: boolean;
}

export const getAdreDeployment = async (): Promise<AdreDeployment> => {
  const res = await api.get<AdreDeployment>('/adre/deployment');
  return res.data;
};

export const updateAdreDeploymentConfig = async (configYaml: string): Promise<void> => {
  await api.put('/adre/deployment/config', { configYaml });
};

export const updateAdreDeploymentModels = async (
  models: AdreDeploymentModelInput[]
): Promise<void> => {
  await api.put('/adre/deployment/models', { models });
};

export const deleteAdreDeploymentModel = async (name: string): Promise<void> => {
  await api.delete(`/adre/deployment/models/${encodeURIComponent(name)}`);
};

export const updateAdreDeploymentPmmUrl = async (pmmUrl: string): Promise<void> => {
  await api.put('/adre/deployment/provisioning', { pmmUrl });
};

export const upsertAdreDeploymentSkill = async (
  skill: AdreDeploymentSkillInput
): Promise<void> => {
  await api.put(`/adre/deployment/skills/${encodeURIComponent(skill.name)}`, skill);
};

export const deleteAdreDeploymentSkill = async (name: string): Promise<void> => {
  await api.delete(`/adre/deployment/skills/${encodeURIComponent(name)}`);
};

export const applyAdreDeployment = async (): Promise<{ restartRequired: boolean; message?: string }> => {
  const res = await api.post<{ restartRequired: boolean; message?: string }>('/adre/deployment/apply', {});
  return res.data;
};

export const provisionAdreDeployment = async (): Promise<void> => {
  await api.post('/adre/deployment/provision', {});
};
