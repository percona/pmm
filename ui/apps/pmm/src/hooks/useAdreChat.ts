import { useState, useCallback, useRef, useEffect } from 'react';
import { adreChatStream, getAdreAlerts, type AdreStreamProgressEvent } from 'api/adre';
import { useAdreSettings } from 'hooks/api/useAdre';
import { useSnackbar } from 'notistack';
import { clearPanelImageCache } from 'components/adre/adre-chat-markdown';
import { PMM_BASE_PATH, PMM_NEW_NAV_GRAFANA_PATH } from 'lib/constants';
import { compactAdreAlertsForToolResult } from 'utils/adreAlertsCompact';
import { PMM_ADRE_FRONTEND_TOOLS } from 'utils/adreFrontendTools';
import { stripQanServiceId } from 'utils/qanServiceId';

const STORAGE_KEY = 'pmm-adre-chat';
const CHAT_HISTORY_WINDOW_MS = 24 * 60 * 60 * 1000;
/** Align with pmm-managed AdreMaxConversationMessagesDefault (context overflow guard). */
const CHAT_HISTORY_MAX_MESSAGES = 40;

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
  mode?: 'fast' | 'investigation' | 'chat';
  dashboardContext?: string;
}

export function useAdreChat() {
  const { data: settings } = useAdreSettings();
  const { enqueueSnackbar } = useSnackbar();
  const [response, setResponse] = useState(() => loadFromStorage().response);
  const [reasoning, setReasoning] = useState(() => loadFromStorage().reasoning);
  const [loading, setLoading] = useState(false);
  const [chatError, setChatError] = useState<string | null>(null);
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
    setChatError(null);
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
      const modeRaw = options?.mode;
      const mode: 'fast' | 'investigation' | undefined =
        modeRaw === 'investigation'
          ? 'investigation'
          : modeRaw === 'chat' || modeRaw === 'fast'
            ? 'fast'
            : undefined;
      const req = {
        ask: userAsk,
        conversation_history: [
          { role: 'system', content: holmesSystemStub },
          ...windowed.map((m: ChatMessage) => ({ role: m.role, content: m.content })),
          { role: 'user', content: userAsk },
        ],
        model: options?.model || undefined,
        stream: true,
        mode,
        frontend_tools: PMM_ADRE_FRONTEND_TOOLS,
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
        onFrontendToolsRequired: async ({ pending_frontend_tool_calls }) => {
          const results: Array<{ tool_call_id: string; tool_name: string; result: string }> = [];
          for (const call of pending_frontend_tool_calls) {
            const result = await executeFrontendTool(call.tool_name, call.arguments ?? {});
            results.push({
              tool_call_id: call.tool_call_id,
              tool_name: call.tool_name,
              result: JSON.stringify(result),
            });
          }

          return results;
        },
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
      const rawMessage = err instanceof Error ? err.message : 'Chat request failed';
      const normalizedMessage = normalizeChatError(rawMessage);
      setChatError(normalizedMessage);
      enqueueSnackbar(normalizedMessage, { variant: 'error' });
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
    setChatError(null);
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
    chatError,
    handleSend,
    clearHistory,
  };
}

/** Pre-`pmm_ui_` names still accepted if Holmes uses an older tool list. */
const LEGACY_PMM_FRONTEND_TOOLS: Record<string, string> = {
  navigate_to_dashboard: 'pmm_ui_navigate_to_dashboard',
  open_explore: 'pmm_ui_open_explore',
  open_investigation: 'pmm_ui_open_investigation',
  focus_qan_query: 'pmm_ui_focus_qan_query',
  open_servicenow_ticket: 'pmm_ui_open_servicenow_ticket',
  check_alerts: 'pmm_ui_check_alerts',
  render_graph: 'pmm_ui_render_graph',
};

function resolvePmmFrontendToolName(name: string): string {
  return LEGACY_PMM_FRONTEND_TOOLS[name] ?? name;
}

/** If the model already URL-encoded Explore `left`, avoid double-encoding (% → %25). */
function exploreLeftQueryParam(raw: string): string {
  const s = String(raw ?? '');
  if (!s) return '';
  if (/%[0-9A-Fa-f]{2}/.test(s)) return s;
  return encodeURIComponent(s);
}

/** Holmes may emit snake_case arguments; handlers expect camelCase. */
function normalizeFrontendToolArgs(raw: Record<string, unknown>): Record<string, unknown> {
  const a = { ...raw };
  const copy = (from: string, to: string) => {
    if (a[to] == null && a[from] != null) a[to] = a[from];
  };
  copy('service_id', 'serviceId');
  copy('query_id', 'queryId');
  copy('dashboard_uid', 'dashboardUid');
  copy('panel_id', 'panelId');
  copy('ticket_id', 'ticketId');
  copy('instance_url', 'instanceUrl');
  copy('investigation_id', 'investigationId');
  return a;
}

async function executeFrontendTool(
  toolName: string,
  args: Record<string, unknown>
): Promise<Record<string, unknown>> {
  const argsNorm = normalizeFrontendToolArgs(args);
  const audit = (outcome: 'success' | 'denied' | 'error', details?: Record<string, unknown>) => {
    try {
      const key = 'pmm-adre-frontend-tool-audit';
      const raw = localStorage.getItem(key);
      const arr = raw ? (JSON.parse(raw) as Array<Record<string, unknown>>) : [];
      arr.push({
        ts: new Date().toISOString(),
        tool: toolName,
        outcome,
        args_hash: hashString(JSON.stringify(argsNorm)),
        ...details,
      });
      localStorage.setItem(key, JSON.stringify(arr.slice(-200)));
    } catch {
      // ignore audit persistence failures
    }
  };

  try {
    const resolvedTool = resolvePmmFrontendToolName(toolName);
    switch (resolvedTool) {
      case 'pmm_ui_navigate_to_dashboard': {
        const uid = String(argsNorm.uid ?? '').trim();
        if (!uid) return { ok: false, error: 'uid is required' };
        const params = new URLSearchParams();
        if (argsNorm.from) params.set('from', String(argsNorm.from));
        if (argsNorm.to) params.set('to', String(argsNorm.to));
        const vars =
          argsNorm.vars && typeof argsNorm.vars === 'object'
            ? (argsNorm.vars as Record<string, unknown>)
            : {};
        Object.entries(vars).forEach(([k, v]) => params.set(`var-${k}`, String(v)));
        const q = params.toString();
        window.open(`/graph/d/${uid}${q ? `?${q}` : ''}`, '_self');
        audit('success');
        return { ok: true };
      }
      case 'pmm_ui_open_explore': {
        const left = exploreLeftQueryParam(String(argsNorm.query ?? ''));
        window.open(`/graph/explore?left=${left}`, '_self');
        audit('success');
        return { ok: true };
      }
      case 'pmm_ui_open_investigation': {
        const id = String(argsNorm.id ?? '').trim();
        if (!id) return { ok: false, error: 'id is required' };
        window.open(`${PMM_BASE_PATH}/investigations/${encodeURIComponent(id)}`, '_self');
        audit('success');
        return { ok: true };
      }
      case 'pmm_ui_focus_qan_query': {
        const serviceId = stripQanServiceId(String(argsNorm.serviceId ?? ''));
        const queryId = String(argsNorm.queryId ?? '');
        window.open(
          `${PMM_BASE_PATH}/qan/ai-insights?service_id=${encodeURIComponent(serviceId)}&query_id=${encodeURIComponent(queryId)}`,
          '_self'
        );
        audit('success');
        return { ok: true };
      }
      case 'pmm_ui_open_servicenow_ticket': {
        const directUrl = String(argsNorm.url ?? '').trim();
        const ticketId = String(argsNorm.ticketId ?? '').trim();
        const instanceUrl = String(argsNorm.instanceUrl ?? '').trim();
        const approved = window.confirm('AI requested opening/creating a ServiceNow ticket. Continue?');
        if (!approved) {
          audit('denied');
          return { ok: false, error: 'user denied action' };
        }
        if (directUrl) {
          window.open(directUrl, '_blank', 'noopener,noreferrer');
          audit('success', { mode: 'direct_url' });
          return { ok: true };
        }
        if (ticketId && instanceUrl) {
          const base = instanceUrl.replace(/\/+$/, '');
          const snURL = `${base}/nav_to.do?uri=incident.do?sys_id=${encodeURIComponent(ticketId)}`;
          window.open(snURL, '_blank', 'noopener,noreferrer');
          audit('success', { mode: 'instance_ticket' });
          return { ok: true };
        }
        const invID = String(argsNorm.investigationId ?? '').trim();
        if (!invID) {
          audit('error', { error: 'missing URL/ticketId context' });
          return { ok: false, error: 'url or (ticketId + instanceUrl) is required' };
        }
        window.open(`${PMM_BASE_PATH}/investigations/${encodeURIComponent(invID)}`, '_self');
        audit('success', { mode: 'fallback_investigation' });
        return { ok: true };
      }
      case 'pmm_ui_check_alerts': {
        const raw = await getAdreAlerts();
        const { value, truncated } = compactAdreAlertsForToolResult(raw);
        audit('success', { truncated });
        return { ok: true, alerts: value, ...(truncated && { truncated: true }) };
      }
      case 'pmm_ui_render_graph': {
        const panelId = String(argsNorm.panelId ?? '').trim();
        const dashboardUID = String(argsNorm.dashboardUid ?? '').trim();
        if (panelId && dashboardUID) {
          const params = new URLSearchParams();
          if (argsNorm.from) params.set('from', String(argsNorm.from));
          if (argsNorm.to) params.set('to', String(argsNorm.to));
          window.open(
            `${PMM_NEW_NAV_GRAFANA_PATH}/d/${encodeURIComponent(dashboardUID)}?viewPanel=${encodeURIComponent(panelId)}${params.toString() ? `&${params.toString()}` : ''}`,
            '_self'
          );
          audit('success', { rendered: true, mode: 'panel_focus' });
          return { ok: true, rendered: true };
        }
        audit('success', { rendered: false });
        return { ok: true, rendered: false, reason: 'Missing dashboardUid/panelId for graph rendering' };
      }
      default:
        audit('error', { error: 'unknown tool' });
        return { ok: false, error: `unknown frontend tool: ${toolName} (resolved: ${resolvedTool})` };
    }
  } catch (e) {
    audit('error', { error: e instanceof Error ? e.message : 'execution failed' });
    return { ok: false, error: e instanceof Error ? e.message : 'execution failed' };
  }
}

function hashString(input: string): string {
  let h = 5381;
  for (let i = 0; i < input.length; i++) {
    h = (h * 33) ^ input.charCodeAt(i);
  }
  return `h${(h >>> 0).toString(16)}`;
}

function normalizeChatError(message: string): string {
  const text = message.toLowerCase();
  if (
    text.includes('token') ||
    text.includes('context window') ||
    text.includes('too large to return') ||
    text.includes('maximum allowed tokens')
  ) {
    return 'Token/context limit reached. Narrow scope (service/time window), reduce Prometheus range/max_points, or retry in smaller steps.';
  }

  return message;
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
