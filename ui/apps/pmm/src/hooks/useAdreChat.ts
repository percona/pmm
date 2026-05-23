import { useState, useCallback, useRef, useEffect } from 'react';
import {
  adreChatStream,
  getAdreAlerts,
  createAdreConversation,
  deleteAdreConversation,
  listAdreConversations,
  getAdreMessages,
  searchAdreMessages,
  normalizeHolmesUsage,
  type AdreStreamProgressEvent,
  type AdreConversation,
  type AdreSearchHit,
  type AdreMessageRow,
  type HolmesUsage,
} from 'api/adre';
import { useAdreSettings } from 'hooks/api/useAdre';
import { useSnackbar } from 'notistack';
import { clearPanelImageCache } from 'components/adre/adre-chat-markdown.utils';
import { PMM_BASE_PATH, PMM_NEW_NAV_GRAFANA_PATH } from 'lib/constants';
import { compactAdreAlertsForToolResult } from 'utils/adreAlertsCompact';
import { PMM_ADRE_FRONTEND_TOOLS } from 'utils/adreFrontendTools';
import { stripQanServiceId } from 'utils/qanServiceId';

/** Resume active conversation after navigation (not legacy chat import). */
const SESSION_CONV_ID_KEY = 'pmm-adre-active-conversation-id';

export type ProgressStep = { id: string; toolName: string; description?: string; status: 'running' | 'done' };

export interface ChatMessage {
  role: 'user' | 'assistant';
  content: string;
  timestamp?: number;
  reasoning?: string;
  progressSteps?: ProgressStep[];
  /** Persisted message id from PMM API (for scroll-to from search). */
  serverMessageId?: number;
  usage?: HolmesUsage;
}

function mapServerRowsToChat(messages: AdreMessageRow[]): ChatMessage[] {
  const out: ChatMessage[] = [];
  for (const m of messages) {
    if (m.role !== 'user' && m.role !== 'assistant') continue;
    const usage = normalizeHolmesUsage(m);
    const hasUsage =
      usage.totalTokens != null || usage.totalCost != null || (usage.model != null && usage.model !== '');
    out.push({
      role: m.role as 'user' | 'assistant',
      content: m.content,
      timestamp: new Date(m.createdAt).getTime(),
      serverMessageId: m.id,
      usage: hasUsage ? usage : undefined,
    });
  }
  return out;
}

export interface SendOptions {
  model?: string;
  mode?: 'fast' | 'investigation' | 'chat';
  dashboardContext?: string;
}

export function useAdreChat() {
  const { data: settings } = useAdreSettings();
  const { enqueueSnackbar } = useSnackbar();
  const [response, setResponse] = useState('');
  const [reasoning, setReasoning] = useState('');
  const [loading, setLoading] = useState(false);
  const [chatError, setChatError] = useState<string | null>(null);
  const [progressSteps, setProgressSteps] = useState<ProgressStep[]>([]);
  const [history, setHistory] = useState<ChatMessage[]>([]);
  const [conversationId, setConversationId] = useState<number | null>(null);
  const [conversations, setConversations] = useState<AdreConversation[]>([]);
  const [conversationsLoading, setConversationsLoading] = useState(false);
  const [searchHits, setSearchHits] = useState<AdreSearchHit[]>([]);
  const [searchLoading, setSearchLoading] = useState(false);
  /** After loading history, AdreChatPanel scrolls to this server message id, then clears. */
  const [scrollToMessageId, setScrollToMessageId] = useState<number | null>(null);
  const streamStartTimeRef = useRef<number | null>(null);
  const progressStepsRef = useRef<ProgressStep[]>([]);

  const refreshConversations = useCallback(async () => {
    setConversationsLoading(true);
    try {
      const { conversations: rows } = await listAdreConversations({ limit: 50 });
      setConversations(rows ?? []);
    } catch {
      /* ignore list errors */
    } finally {
      setConversationsLoading(false);
    }
  }, []);

  const loadMessagesFor = useCallback(async (id: number) => {
    const { messages } = await getAdreMessages(id, { limit: 100 });
    setHistory(mapServerRowsToChat(messages));
  }, []);

  const selectConversation = useCallback(
    async (id: number, options?: { focusMessageId?: number }) => {
      if (typeof id !== 'number' || Number.isNaN(id)) {
        enqueueSnackbar('Invalid conversation', { variant: 'error' });
        return;
      }
      setScrollToMessageId(options?.focusMessageId ?? null);
      setConversationId(id);
      try {
        sessionStorage.setItem(SESSION_CONV_ID_KEY, String(id));
      } catch {
        /* ignore */
      }
      setResponse('');
      setReasoning('');
      setChatError(null);
      try {
        await loadMessagesFor(id);
      } catch {
        enqueueSnackbar('Failed to load messages', { variant: 'error' });
        setScrollToMessageId(null);
      }
    },
    [loadMessagesFor, enqueueSnackbar]
  );

  const clearScrollToMessage = useCallback(() => setScrollToMessageId(null), []);

  const newChat = useCallback(async () => {
    try {
      const c = await createAdreConversation();
      setConversationId(c.id);
      try {
        sessionStorage.setItem(SESSION_CONV_ID_KEY, String(c.id));
      } catch {
        /* ignore */
      }
      setHistory([]);
      setResponse('');
      setReasoning('');
      setChatError(null);
      await refreshConversations();
    } catch (e) {
      enqueueSnackbar(e instanceof Error ? e.message : 'Failed to start chat', { variant: 'error' });
    }
  }, [enqueueSnackbar, refreshConversations]);

  const deleteConversation = useCallback(
    async (id: number) => {
      if (!window.confirm('Delete this conversation? This cannot be undone.')) return;
      try {
        await deleteAdreConversation(id);
        await refreshConversations();
        if (conversationId === id) {
          try {
            sessionStorage.removeItem(SESSION_CONV_ID_KEY);
          } catch {
            /* ignore */
          }
          await newChat();
        }
      } catch (e) {
        enqueueSnackbar(e instanceof Error ? e.message : 'Failed to delete conversation', { variant: 'error' });
      }
    },
    [conversationId, enqueueSnackbar, refreshConversations, newChat]
  );

  /** Invalidates stale async work when the effect re-runs (e.g. React Strict Mode remount or ADRE disabled). */
  const adreInitGenRef = useRef(0);
  const settingsEnabled = settings?.enabled ?? false;
  const settingsLoaded = settings !== undefined;
  useEffect(() => {
    if (!settingsEnabled) {
      if (settingsLoaded) {
        adreInitGenRef.current++;
      }
      return;
    }
    const gen = ++adreInitGenRef.current;
    (async () => {
      try {
        const raw = sessionStorage.getItem(SESSION_CONV_ID_KEY);
        if (raw) {
          const id = parseInt(raw, 10);
          if (!Number.isNaN(id)) {
            await selectConversation(id);
            if (gen !== adreInitGenRef.current) return;
            await refreshConversations();
            return;
          }
        }
        const c = await createAdreConversation();
        if (gen !== adreInitGenRef.current) return;
        setConversationId(c.id);
        try {
          sessionStorage.setItem(SESSION_CONV_ID_KEY, String(c.id));
        } catch {
          /* ignore */
        }
        setHistory([]);
        await refreshConversations();
        if (gen !== adreInitGenRef.current) return;
      } catch {
        if (gen !== adreInitGenRef.current) return;
        enqueueSnackbar('Failed to initialize chat', { variant: 'error' });
      }
    })();
  }, [settingsEnabled, settingsLoaded, selectConversation, refreshConversations, enqueueSnackbar]);

  const runSearch = useCallback(
    async (q: string) => {
      const t = q.trim();
      if (!t) {
        setSearchHits([]);
        setSearchLoading(false);
        return;
      }
      setSearchLoading(true);
      try {
        const { hits } = await searchAdreMessages(t, 30);
        setSearchHits(hits ?? []);
      } catch {
        enqueueSnackbar('Search failed', { variant: 'error' });
      } finally {
        setSearchLoading(false);
      }
    },
    [enqueueSnackbar]
  );

  const handleSend = useCallback(
    async (ask: string, options?: SendOptions) => {
      const userAsk = ask.trim();
      if (!userAsk) return;
      if (conversationId == null) {
        enqueueSnackbar('No active conversation', { variant: 'warning' });
        return;
      }

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
        const modeRaw = options?.mode;
        const mode: 'fast' | 'investigation' | undefined =
          modeRaw === 'investigation'
            ? 'investigation'
            : modeRaw === 'chat' || modeRaw === 'fast'
              ? 'fast'
              : undefined;
        const req = {
          ask: userAsk,
          conversation_id: conversationId,
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
            const next = [
              ...progressStepsRef.current,
              { id: event.id, toolName: event.toolName, description: event.description, status: 'running' as const },
            ];
            progressStepsRef.current = next;
            setProgressSteps(next);
          } else {
            const next = progressStepsRef.current.map((s: ProgressStep) =>
              s.id === event.id ? { ...s, status: 'done' as const } : s
            );
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
        void refreshConversations();
      } catch (err) {
        const rawMessage = err instanceof Error ? err.message : 'Chat request failed';
        const normalizedMessage = normalizeChatError(rawMessage);
        setChatError(normalizedMessage);
        enqueueSnackbar(normalizedMessage, { variant: 'error' });
        try {
          await loadMessagesFor(conversationId);
        } catch {
          /* ignore */
        }
      } finally {
        setLoading(false);
        setProgressSteps([]);
        progressStepsRef.current = [];
        streamStartTimeRef.current = null;
      }
    },
    [conversationId, enqueueSnackbar, refreshConversations, loadMessagesFor]
  );

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
    void newChat();
    clearPanelImageCache();
  }, [newChat]);

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
    conversationId,
    conversations,
    conversationsLoading,
    refreshConversations,
    newChat,
    deleteConversation,
    selectConversation,
    scrollToMessageId,
    clearScrollToMessage,
    searchHits,
    searchLoading,
    runSearch,
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
