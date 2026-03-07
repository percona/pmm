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

export const adreChatStream = async (
  body: AdreChatRequest,
  onChunk: (chunk: string) => void
): Promise<void> => {
  // Relative URL: assumes same-origin or proxy so /v1/adre/chat is served by PMM backend
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
          const text = extractTextFromSSEData(data);
          if (text) onChunk(text);
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
          const text = extractTextFromSSEData(data);
          if (text) onChunk(text);
        }
      }
    }
  }
};

/** Parses SSE data line; if JSON with text/delta/content/analysis, returns that string, else returns raw. */
function extractTextFromSSEData(data: string): string {
  const trimmed = data.trim();
  if (!trimmed || trimmed === '[DONE]') return '';
  if (trimmed.startsWith('{')) {
    try {
      const o = JSON.parse(trimmed) as Record<string, unknown>;
      const raw = o.text ?? o.delta ?? o.content ?? o.analysis;
      if (typeof raw === 'string') return raw;
      // delta may be an object with content (e.g. { content: "..." })
      if (raw && typeof raw === 'object' && 'content' in raw && typeof (raw as { content: unknown }).content === 'string') {
        return (raw as { content: string }).content;
      }
    } catch {
      // not JSON or invalid
    }
  }
  return trimmed;
}
