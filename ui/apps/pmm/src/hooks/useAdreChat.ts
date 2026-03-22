import { useState, useCallback, useRef, useEffect } from 'react';
import { adreChatStream, type AdreStreamProgressEvent } from 'api/adre';
import { useAdreSettings } from 'hooks/api/useAdre';
import { useSnackbar } from 'notistack';
import { clearPanelImageCache } from 'components/adre/adre-chat-markdown';

const STORAGE_KEY = 'pmm-adre-chat';
const CHAT_HISTORY_WINDOW_MS = 24 * 60 * 60 * 1000;
const CHAT_HISTORY_MAX_MESSAGES = 30;

export type ProgressStep = { id: string; toolName: string; description?: string; status: 'running' | 'done' };

export interface ChatMessage {
  role: 'user' | 'assistant';
  content: string;
  timestamp?: number;
  reasoning?: string;
  progressSteps?: ProgressStep[];
}

function isValidProgressStep(s: unknown): s is ProgressStep {
  return (
    typeof s === 'object' &&
    s != null &&
    typeof (s as ProgressStep).id === 'string' &&
    typeof (s as ProgressStep).toolName === 'string' &&
    ((s as ProgressStep).description === undefined || typeof (s as ProgressStep).description === 'string') &&
    ((s as ProgressStep).status === 'running' || (s as ProgressStep).status === 'done')
  );
}

function getWindowedHistory(history: ChatMessage[]): ChatMessage[] {
  if (history.length === 0) return [];
  const newestTs = Math.max(...history.map((m) => m.timestamp ?? 0));
  const cutoff = newestTs - CHAT_HISTORY_WINDOW_MS;
  const windowed = history.filter((m) => (m.timestamp ?? 0) >= cutoff);
  if (windowed.length <= CHAT_HISTORY_MAX_MESSAGES) return windowed;

  return windowed.slice(-CHAT_HISTORY_MAX_MESSAGES);
}

function loadFromStorage(): { response: string; reasoning: string; history: ChatMessage[] } {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (raw) {
      const parsed = JSON.parse(raw) as {
        response?: string;
        reasoning?: string;
        history?: unknown[];
      };
      const rawHistory = Array.isArray(parsed.history)
        ? (parsed.history as unknown[]).filter((m): m is ChatMessage => {
            if (!m || typeof m !== 'object' || typeof (m as ChatMessage).content !== 'string') return false;
            const role = (m as ChatMessage).role;
            if (role !== 'user' && role !== 'assistant') return false;
            const steps = (m as ChatMessage).progressSteps;
            if (steps !== undefined && (!Array.isArray(steps) || !steps.every(isValidProgressStep))) return false;

            return true;
          })
        : [];
      const normalizedHistory = rawHistory.map((m) => {
        if (m.progressSteps?.length) {
          const steps = m.progressSteps.filter(isValidProgressStep);

          return { ...m, progressSteps: steps.length > 0 ? steps : undefined };
        }

        return m;
      });
      const history = getWindowedHistory(normalizedHistory);

      return {
        response: typeof parsed.response === 'string' ? parsed.response : '',
        reasoning: typeof parsed.reasoning === 'string' ? parsed.reasoning : '',
        history,
      };
    }
  } catch {
    // ignore
  }

  return { response: '', reasoning: '', history: [] };
}

function saveToStorage(response: string, reasoning: string, history: ChatMessage[]) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify({ response, reasoning, history }));
  } catch {
    // ignore
  }
}

function persistAssistantToHistory(
  userContent: string,
  assistantContent: string,
  assistantReasoning: string,
  progressSteps: ProgressStep[] = []
): void {
  const { history } = loadFromStorage();
  const last = history[history.length - 1];
  const hasUserMsg = last?.role === 'user' && last?.content === userContent;
  const assistantMsg: ChatMessage = {
    role: 'assistant',
    content: assistantContent,
    timestamp: Date.now(),
    reasoning: assistantReasoning || undefined,
    ...(progressSteps.length > 0 && { progressSteps }),
  };
  const toAppend: ChatMessage[] = hasUserMsg
    ? [assistantMsg]
    : [{ role: 'user', content: userContent, timestamp: Date.now() }, assistantMsg];
  const updatedHistory = [...history, ...toAppend];
  const windowed = getWindowedHistory(updatedHistory);
  saveToStorage('', '', windowed);
}

export interface SendOptions {
  model?: string;
  mode?: 'chat' | 'investigation';
  dashboardContext?: string;
}

export function useAdreChat() {
  const { data: settings } = useAdreSettings();
  const { enqueueSnackbar } = useSnackbar();
  const [response, setResponse] = useState(() => loadFromStorage().response);
  const [reasoning, setReasoning] = useState(() => loadFromStorage().reasoning);
  const [loading, setLoading] = useState(false);
  const [progressSteps, setProgressSteps] = useState<ProgressStep[]>([]);
  const [history, setHistory] = useState<ChatMessage[]>(() => loadFromStorage().history);
  const streamStartTimeRef = useRef<number | null>(null);
  const progressStepsRef = useRef<ProgressStep[]>([]);

  useEffect(() => {
    saveToStorage(response, reasoning, history);
  }, [response, reasoning, history]);

  const handleSend = useCallback(async (ask: string, options?: SendOptions) => {
    const userAsk = ask.trim();
    if (!userAsk) return;

    setLoading(true);
    setResponse('');
    setReasoning('');
    setProgressSteps([]);
    progressStepsRef.current = [];
    streamStartTimeRef.current = Date.now();

    const userTimestamp = Date.now();
    setHistory((prev: ChatMessage[]) => [...prev, { role: 'user', content: userAsk, timestamp: userTimestamp }]);

    try {
      const windowed = getWindowedHistory(history);
      // Grafana context: pmm-managed appends dashboard_context to Holmes additional_system_prompt (authoritative for current panel).
      // HolmesGPT still requires conversation_history[0].role === 'system' (Pydantic ChatRequest); use a short placeholder — not the full Grafana blob.
      const holmesSystemStub =
        'You are assisting a PMM user. The server supplies full system instructions and any current Grafana page context via additional_system_prompt.';
      const req = {
        ask: userAsk,
        conversation_history: [
          { role: 'system', content: holmesSystemStub },
          ...windowed.map((m: ChatMessage) => ({ role: m.role, content: m.content })),
          { role: 'user', content: userAsk },
        ],
        model: options?.model || undefined,
        stream: true,
        mode: options?.mode,
        ...(options?.dashboardContext?.trim()
          ? { dashboard_context: options.dashboardContext.trim() }
          : {}),
      };

      let fullResponse = '';
      let fullReasoning = '';
      const handleProgress = (event: AdreStreamProgressEvent) => {
        if (event.type === 'start_tool') {
          const next = [...progressStepsRef.current, { id: event.id, toolName: event.toolName, description: event.description, status: 'running' as const }];
          progressStepsRef.current = next;
          setProgressSteps(next);
        } else {
          const next = progressStepsRef.current.map((s: ProgressStep) => (s.id === event.id ? { ...s, status: 'done' as const } : s));
          progressStepsRef.current = next;
          setProgressSteps(next);
        }
      };

      await adreChatStream(req, {
        onChunk: (contentChunk, reasoningChunk) => {
          if (contentChunk) fullResponse += contentChunk;
          if (reasoningChunk) fullReasoning += reasoningChunk;
          setReasoning(fullReasoning);
          setResponse(fullResponse);
        },
        onProgress: handleProgress,
      });

      const finalProgressSteps = progressStepsRef.current;
      persistAssistantToHistory(userAsk, fullResponse, fullReasoning, finalProgressSteps);
      setHistory((prev: ChatMessage[]) => [
        ...prev,
        {
          role: 'assistant',
          content: fullResponse,
          timestamp: Date.now(),
          reasoning: fullReasoning || undefined,
          ...(finalProgressSteps.length > 0 && { progressSteps: finalProgressSteps }),
        },
      ]);
      setResponse('');
      setReasoning('');
    } catch (err) {
      enqueueSnackbar(err instanceof Error ? err.message : 'Chat request failed', { variant: 'error' });
    } finally {
      setLoading(false);
      setProgressSteps([]);
      progressStepsRef.current = [];
      streamStartTimeRef.current = null;
    }
  }, [history, enqueueSnackbar]);

  const allMessages: (ChatMessage & { streaming?: boolean })[] = [
    ...history,
    ...(response || reasoning || loading
      ? [
          {
            role: 'assistant' as const,
            content: response,
            timestamp: streamStartTimeRef.current ?? Date.now(),
            reasoning: reasoning || undefined,
            streaming: true,
          },
        ]
      : []),
  ];

  const clearHistory = useCallback(() => {
    setHistory([]);
    setResponse('');
    setReasoning('');
    setProgressSteps([]);
    progressStepsRef.current = [];
    clearPanelImageCache();
    try {
      localStorage.removeItem(STORAGE_KEY);
    } catch { /* ignore */ }
  }, []);

  return {
    history,
    response,
    reasoning,
    loading,
    progressSteps,
    allMessages,
    settings,
    handleSend,
    clearHistory,
  };
}

export function formatTimestamp(ts: number): string {
  const d = new Date(ts);
  const now = Date.now();
  const diff = now - ts;
  if (diff < 60_000) return 'Just now';
  if (diff < 3600_000) return `${Math.floor(diff / 60_000)}m ago`;
  if (diff < 86400_000) return d.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' });

  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' });
}
