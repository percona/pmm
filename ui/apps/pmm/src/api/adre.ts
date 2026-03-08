import { api } from './api';

export interface AdreSettings {
  enabled: boolean;
  url: string;
}

export interface AdreModelsResponse {
  modelName: string[];
}

export interface AdreChatRequest {
  ask: string;
  conversationHistory?: unknown[];
  model?: string;
  stream?: boolean;
  additionalSystemPrompt?: string;
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

export const adreChatStream = async (
  body: AdreChatRequest,
  onChunk: AdreChatStreamCallback
): Promise<void> => {
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

export const adreInvestigateStream = async (
  body: AdreInvestigateRequest,
  onChunk: (chunk: string) => void
): Promise<void> => {
  // Same as chat: relative /v1/adre/investigate
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
      const contentRaw = o.text ?? o.delta ?? o.content ?? o.analysis;
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

function stringFromValue(v: unknown): string | undefined {
  if (typeof v === 'string') return v;
  if (v && typeof v === 'object' && 'content' in v && typeof (v as { content: unknown }).content === 'string') {
    return (v as { content: string }).content;
  }
  return undefined;
}
