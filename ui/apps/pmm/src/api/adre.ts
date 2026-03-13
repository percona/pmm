import { api } from './api';

export interface AdreSettings {
  enabled: boolean;
  url: string;
  chatPrompt?: string;
  investigationPrompt?: string;
  /** Display value when chat_prompt is empty (built-in default). */
  chatPromptDisplay?: string;
  /** Display value when investigation_prompt is empty (built-in default). */
  investigationPromptDisplay?: string;
  defaultChatMode?: 'chat' | 'investigation';
  /** "holmesgpt" = Holmes Agent (direct); "holmes_agent" = PMM Agent (Holmes with replace_system_prompt). */
  chatBackend?: 'holmesgpt' | 'holmes_agent';
  /** Max messages sent to PMM Agent (5–100). Used when chatBackend is holmes_agent. */
  chatHistoryLength?: number;
  /** System prompt for PMM Agent when chatBackend is holmes_agent. Empty = use built-in default. */
  agentPrompt?: string;
  /** Display value when agent_prompt is empty (built-in default). */
  agentPromptDisplay?: string;
  /** Backend may return snake_case; frontend may use either. */
  chat_history_length?: number;
}

export interface AdreModelsResponse {
  modelName: string[];
}

export interface AdreChatRequest {
  ask: string;
  conversation_history?: unknown[];
  model?: string;
  stream?: boolean;
  /** Server resolves prompt from mode; client must not send additionalSystemPrompt. */
  mode?: 'chat' | 'investigation';
  pageContext?: unknown;
}

export interface AdreChatResponse {
  analysis: string;
  conversationHistory?: unknown[];
  toolCalls?: unknown[];
  followUpActions?: unknown[];
}

export interface AdreInvestigateRequest {
  source: string;
  title: string;
  description: string;
  subject?: unknown;
  context?: unknown;
  model?: string;
  stream?: boolean;
}

export interface AdreInvestigateResponse {
  analysis: string;
  sections?: Record<string, string>;
  toolCalls?: unknown[];
  instructions?: unknown[];
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

export const adreInvestigate = async (
  body: AdreInvestigateRequest
): Promise<AdreInvestigateResponse> => {
  const res = await api.post<AdreInvestigateResponse>('/adre/investigate', body);
  return res.data;
};

/** Callback for adreInvestigateStream: receives content chunks and/or reasoning chunks. */
export type AdreInvestigateStreamCallback = (content?: string, reasoning?: string) => void;

export const adreInvestigateStream = async (
  body: AdreInvestigateRequest,
  onChunk: AdreInvestigateStreamCallback
): Promise<void> => {
  const response = await fetch('/v1/adre/investigate', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify({ ...body, stream: true }),
  });
  if (!response.ok) {
    const err = await response.json().catch(() => ({}));
    throw new Error(err.error || `Investigate failed: ${response.status}`);
  }
  const reader = response.body?.getReader();
  if (!reader) throw new Error('No response body');
  const decoder = new TextDecoder();
  let buffer = '';
  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    buffer += decoder.decode(value, { stream: true });
    const lines = buffer.split('\n');
    buffer = lines.pop() || '';
    for (const line of lines) {
      if (line.startsWith('data: ')) {
        const data = line.slice(6);
        if (data !== '[DONE]' && data.trim()) {
          const parsed = parseSSEData(data);
          if (parsed.content) onChunk(parsed.content);
          if (parsed.reasoning) onChunk(undefined, parsed.reasoning);
        }
      }
    }
  }
};

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
