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
  'time_runbooks',
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
}

export interface AdreModelsResponse {
  modelName: string[];
}

export interface AdreChatRequest {
  ask: string;
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

export interface AdreChatResponse {
  analysis: string;
  conversationHistory?: unknown[];
  toolCalls?: unknown[];
  followUpActions?: unknown[];
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
  cached?: boolean;
}

export interface AdreCreateServiceNowFromInsightsRequest {
  serviceId: string;
  queryText: string;
  analysis: string;
  queryId?: string;
  fingerprint?: string;
  timeFrom?: string;
  timeTo?: string;
}

export interface AdreCreateServiceNowFromInsightsResponse {
  success: boolean;
  ticket_id: string;
  ticket_number?: string;
  message: string;
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

export const adreChat = async (
  body: AdreChatRequest
): Promise<AdreChatResponse> => {
  const res = await api.post<AdreChatResponse>('/adre/chat', body);
  return res.data;
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

export const createServiceNowFromQanInsights = async (
  body: AdreCreateServiceNowFromInsightsRequest
): Promise<AdreCreateServiceNowFromInsightsResponse> => {
  const res = await api.post<AdreCreateServiceNowFromInsightsResponse>(
    '/adre/qan-insights/servicenow',
    {
      service_id: body.serviceId,
      query_text: body.queryText,
      analysis: body.analysis,
      ...(body.queryId ? { query_id: body.queryId } : {}),
      ...(body.fingerprint ? { fingerprint: body.fingerprint } : {}),
      ...(body.timeFrom ? { time_from: body.timeFrom } : {}),
      ...(body.timeTo ? { time_to: body.timeTo } : {}),
    }
  );
  return res.data;
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
  while (true) {
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
