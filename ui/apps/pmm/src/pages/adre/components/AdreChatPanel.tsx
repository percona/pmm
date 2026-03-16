import {
  Box,
  Collapse,
  IconButton,
  Link,
  MenuItem,
  Select,
  Stack,
  TextField,
  Tooltip,
  Typography,
} from '@mui/material';
import ExpandLess from '@mui/icons-material/ExpandLess';
import ExpandMore from '@mui/icons-material/ExpandMore';
import Send from '@mui/icons-material/Send';
import { FC, useState, useCallback, useEffect, useRef, ReactNode } from 'react';
import Markdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import rehypeRaw from 'rehype-raw';
import { useAdreModels, useAdreSettings } from 'hooks/api/useAdre';
import { adreChatStream, type AdreStreamProgressEvent } from 'api/adre';
import { useSnackbar } from 'notistack';
import { CodeBlock } from 'pages/updates/change-log/code-block';
import { PMM_NEW_NAV_GRAFANA_PATH } from 'lib/constants';

const STORAGE_KEY = 'pmm-adre-chat';
const CHAT_HISTORY_WINDOW_MS = 24 * 60 * 60 * 1000;
const CHAT_HISTORY_MAX_MESSAGES = 300;

export type ProgressStep = { id: string; toolName: string; description?: string; status: 'running' | 'done' };

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

/** Parse ISO 8601 to epoch ms for Grafana dashboard URL; return original string if not parseable. */
function toEpochMsOrOriginal(s: string): string {
  if (!s) return s;
  const date = new Date(s);
  if (Number.isNaN(date.getTime())) return s;
  return String(date.getTime());
}

const GRAFANA_RENDER_PATH = '/v1/grafana/render';
const GRAFANA_RENDER_D_SOLO = '/graph/render/d-solo/';
const RENDER_IMAGE_TIMEOUT_MS = 60000;

/** Build "Open in Grafana" URL from a PMM or Grafana render URL. Uses epoch ms for from/to when possible. */
function dashboardUrlFromRenderUrl(renderSrc: string): string | null {
  try {
    const path = renderSrc.startsWith('/') ? renderSrc : renderSrc.includes('://') ? new URL(renderSrc).pathname : `/${renderSrc}`;
    const searchStart = path.indexOf('?');
    const pathOnly = searchStart === -1 ? path : path.slice(0, searchStart);
    const params = new URLSearchParams(searchStart === -1 ? '' : path.slice(searchStart + 1));

    let uid: string | null = null;
    let panelId: string | null = null;

    if (pathOnly.includes(GRAFANA_RENDER_D_SOLO)) {
      const match = pathOnly.match(/\/graph\/render\/d-solo\/([^/]+)/);
      uid = match ? match[1] : null;
      panelId = params.get('panelId');
    } else {
      uid = params.get('dashboard_uid');
      panelId = params.get('panel_id');
    }

    const from = params.get('from');
    const to = params.get('to');
    if (!uid) return null;
    const base = `${PMM_NEW_NAV_GRAFANA_PATH}/d/${uid}`;
    const q = new URLSearchParams();
    if (panelId) q.set('viewPanel', panelId);
    if (from) q.set('from', toEpochMsOrOriginal(from));
    if (to) q.set('to', toEpochMsOrOriginal(to));
    params.forEach((v, k) => {
      if (k.startsWith('var-')) q.set(k, v);
    });
    const qs = q.toString();
    return qs ? `${base}?${qs}` : base;
  } catch {
    return null;
  }
}

function isGrafanaRenderImageSrc(src: string): boolean {
  if (src.includes(GRAFANA_RENDER_PATH) && src.includes('dashboard_uid=') && src.includes('panel_id=')) return true;
  return src.includes(GRAFANA_RENDER_D_SOLO) && src.includes('panelId=');
}

/** Fetches Grafana render image with credentials and long timeout so the panel image loads in chat. */
const GrafanaPanelImage: FC<{
  src: string;
  alt: string;
  dashboardHref: string | null;
}> = ({ src, alt, dashboardHref }) => {
  const [state, setState] = useState<'loading' | { status: 'success'; url: string } | { status: 'error' }>('loading');

  useEffect(() => {
    let objectUrl: string | null = null;
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), RENDER_IMAGE_TIMEOUT_MS);

    fetch(src, { credentials: 'include', signal: controller.signal })
      .then((res) => {
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        return res.blob();
      })
      .then((blob) => {
        objectUrl = URL.createObjectURL(blob);
        setState({ status: 'success', url: objectUrl });
      })
      .catch(() => setState({ status: 'error' }))
      .finally(() => clearTimeout(timeoutId));

    return () => {
      clearTimeout(timeoutId);
      if (objectUrl) URL.revokeObjectURL(objectUrl);
    };
  }, [src]);

  if (state === 'loading') {
    return (
      <Typography variant="body2" color="text.secondary" sx={{ py: 2 }}>
        Loading panel image…
      </Typography>
    );
  }
  if (state.status === 'error') {
    return (
      <Box sx={{ my: 1 }}>
        <Typography variant="body2" color="text.secondary">
          Image failed to load
        </Typography>
        {dashboardHref && (
          <Link href={dashboardHref} target="_blank" rel="noopener noreferrer" sx={{ display: 'inline-block', mt: 0.5, fontSize: '0.8125rem' }}>
            Open in Grafana
          </Link>
        )}
      </Box>
    );
  }
  return (
    <Box sx={{ my: 1 }}>
      <Box
        component="img"
        src={state.url}
        alt={alt}
        loading="lazy"
        sx={{ maxWidth: '100%', height: 'auto', borderRadius: 1, display: 'block' }}
      />
      {dashboardHref && (
        <Link
          href={dashboardHref}
          target="_blank"
          rel="noopener noreferrer"
          sx={{ display: 'inline-block', mt: 0.5, fontSize: '0.8125rem' }}
        >
          Open in Grafana
        </Link>
      )}
    </Box>
  );
};

interface ChatMessage {
  role: 'user' | 'assistant';
  content: string;
  timestamp?: number;
  reasoning?: string;
  progressSteps?: ProgressStep[];
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

/** Persists the assistant message to localStorage when stream completes. Runs outside React state so it works even if the component unmounts during streaming. */
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

/** Returns history limited to last 24h from the newest message, capped at CHAT_HISTORY_MAX_MESSAGES. */
function getWindowedHistory(history: ChatMessage[]): ChatMessage[] {
  if (history.length === 0) return [];
  const newestTs = Math.max(...history.map((m) => m.timestamp ?? 0));
  const cutoff = newestTs - CHAT_HISTORY_WINDOW_MS;
  const windowed = history.filter((m) => (m.timestamp ?? 0) >= cutoff);
  if (windowed.length <= CHAT_HISTORY_MAX_MESSAGES) return windowed;
  return windowed.slice(-CHAT_HISTORY_MAX_MESSAGES);
}

function formatTimestamp(ts: number): string {
  const d = new Date(ts);
  const now = Date.now();
  const diff = now - ts;
  if (diff < 60_000) return 'Just now';
  if (diff < 3600_000) return `${Math.floor(diff / 60_000)}m ago`;
  if (diff < 86400_000) return d.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' });
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' });
}

export const AdreChatPanel: FC = () => {
  const { data: models = [] } = useAdreModels();
  const { data: settings } = useAdreSettings();
  const { enqueueSnackbar } = useSnackbar();
  const [ask, setAsk] = useState('');
  const [model, setModel] = useState('');
  const [mode, setMode] = useState<'chat' | 'investigation'>('chat');
  const [response, setResponse] = useState(() => loadFromStorage().response);
  const [reasoning, setReasoning] = useState(() => loadFromStorage().reasoning);
  const [loading, setLoading] = useState(false);
  const [progressSteps, setProgressSteps] = useState<Array<{ id: string; toolName: string; description?: string; status: 'running' | 'done' }>>([]);
  const [history, setHistory] = useState<ChatMessage[]>(() => loadFromStorage().history);
  const [expandedReasoningIdx, setExpandedReasoningIdx] = useState<number | null>(null);
  const [expandedProgressIdx, setExpandedProgressIdx] = useState<number | null>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const streamStartTimeRef = useRef<number | null>(null);
  const progressStepsRef = useRef<ProgressStep[]>([]);

  const defaultModeSyncedRef = useRef(false);
  useEffect(() => {
    if (!defaultModeSyncedRef.current && (settings?.defaultChatMode === 'investigation' || settings?.defaultChatMode === 'chat')) {
      defaultModeSyncedRef.current = true;
      setMode(settings.defaultChatMode);
    }
  }, [settings?.defaultChatMode]);

  useEffect(() => {
    saveToStorage(response, reasoning, history);
  }, [response, reasoning, history]);

  const scrollToBottom = useCallback(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, []);

  useEffect(() => {
    scrollToBottom();
  }, [history.length, response, reasoning, scrollToBottom]);

  useEffect(() => {
    const id = requestAnimationFrame(() => {
      messagesEndRef.current?.scrollIntoView({ behavior: 'auto' });
    });
    return () => cancelAnimationFrame(id);
  }, []);

  const handleSend = useCallback(async () => {
    if (!ask.trim()) return;
    const userAsk = ask.trim();
    setLoading(true);
    setResponse('');
    setReasoning('');
    setProgressSteps([]);
    progressStepsRef.current = [];
    streamStartTimeRef.current = Date.now();
    setAsk('');
    const userTimestamp = Date.now();
    setHistory((prev: ChatMessage[]) => [...prev, { role: 'user', content: userAsk, timestamp: userTimestamp }]);
    try {
      const windowed = getWindowedHistory(history);
      const req = {
        ask: userAsk,
        conversation_history: [
          { role: 'system', content: 'You are a helpful AI ops assistant for Percona Monitoring and Management (PMM).' },
          ...windowed.map((m: ChatMessage) => ({ role: m.role, content: m.content })),
          { role: 'user', content: userAsk },
        ],
        model: model || undefined,
        stream: true,
        mode,
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
  }, [ask, history, model, mode, enqueueSnackbar]);

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

  return (
    <Box sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <Stack direction="row" alignItems="center" justifyContent="space-between" flexWrap="wrap" gap={1} sx={{ mb: 1, px: 0 }}>
        <Stack direction="row" alignItems="center" gap={2}>
          <Stack direction="row" sx={{ borderBottom: 1, borderColor: 'divider' }}>
            <Typography
              component="button"
              variant="body2"
              onClick={() => setMode('chat')}
              sx={{
                border: 'none',
                background: 'none',
                cursor: 'pointer',
                p: 1,
                pb: 1.5,
                color: mode === 'chat' ? 'text.primary' : 'text.secondary',
                borderBottom: mode === 'chat' ? 2 : 0,
                borderColor: 'primary.main',
                mb: -0.5,
                borderRadius: 0,
              }}
            >
              Chat
            </Typography>
            <Typography
              component="button"
              variant="body2"
              onClick={() => setMode('investigation')}
              sx={{
                border: 'none',
                background: 'none',
                cursor: 'pointer',
                p: 1,
                pb: 1.5,
                color: mode === 'investigation' ? 'text.primary' : 'text.secondary',
                borderBottom: mode === 'investigation' ? 2 : 0,
                borderColor: 'primary.main',
                mb: -0.5,
                borderRadius: 0,
              }}
            >
              Investigation
            </Typography>
          </Stack>
          <Tooltip
            title={
              settings?.chatBackend === 'holmes_agent' && settings?.url
                ? 'Chat via PMM Agent'
                : 'Chat via Holmes Agent'
            }
          >
            <Typography variant="caption" color="text.secondary" sx={{ cursor: 'help' }}>
              {settings?.chatBackend === 'holmes_agent' && settings?.url ? 'PMM Agent' : 'Holmes Agent'}
            </Typography>
          </Tooltip>
        </Stack>
        <Select
          value={model}
          onChange={(e: { target: { value: string } }) => setModel(e.target.value)}
          size="small"
          displayEmpty
          sx={{ minWidth: 120, fontSize: '0.8125rem' }}
          renderValue={(v) => v || 'Default'}
        >
          <MenuItem value="">Default</MenuItem>
          {models.map((m: string) => (
            <MenuItem key={m} value={m}>
              {m}
            </MenuItem>
          ))}
        </Select>
      </Stack>
        <Stack gap={1} sx={{ flex: 1, minHeight: 0 }}>
          <Box
            ref={containerRef}
            id="messages-container"
            sx={{
              flex: 1,
              minHeight: 280,
              maxHeight: '70vh',
              overflow: 'auto',
              p: 2,
              display: 'flex',
              flexDirection: 'column',
              gap: 2,
              bgcolor: '#212121',
              borderRadius: 1,
            }}
          >
            {allMessages.length === 0 ? (
              <Typography color="text.secondary" variant="body2" sx={{ alignSelf: 'center', mt: 2 }}>
                Ask a question about your database environment...
              </Typography>
            ) : (
              <Box sx={{ maxWidth: '100%', width: '100%', alignSelf: 'center', display: 'flex', flexDirection: 'column', gap: 2 }}>
              {allMessages.map((msg, idx) => (
                <Box
                  key={idx}
                  sx={{
                    display: 'flex',
                    justifyContent: msg.role === 'user' ? 'flex-end' : 'flex-start',
                    alignSelf: msg.role === 'user' ? 'flex-end' : 'flex-start',
                    maxWidth: '85%',
                  }}
                >
                  <Box
                    sx={{
                      px: 2,
                      py: 1.5,
                      borderRadius: 2,
                      ...(msg.role === 'user'
                        ? {
                            bgcolor: '#2d3748',
                            color: 'text.primary',
                          }
                        : {
                            bgcolor: 'rgba(255,255,255,0.05)',
                            border: 1,
                            borderColor: 'rgba(255,255,255,0.12)',
                          }),
                    }}
                  >
                    <Typography
                      variant="caption"
                      color={msg.role === 'user' ? 'text.secondary' : 'text.secondary'}
                      display="block"
                      sx={{ mb: 0.5, fontSize: '0.7rem', opacity: 0.8 }}
                    >
                      {msg.role === 'user' ? 'You' : 'Assistant'}
                      {msg.timestamp ? ` · ${formatTimestamp(msg.timestamp)}` : ''}
                    </Typography>
                    {msg.role === 'user' ? (
                      <Typography sx={{ whiteSpace: 'pre-wrap' }}>{msg.content}</Typography>
                    ) : (
                      <Box>
                        {(msg.reasoning ?? (msg.streaming && reasoning)) && (
                          <>
                            <IconButton
                              size="small"
                              onClick={() => setExpandedReasoningIdx((prev: number | null) => (prev === idx ? null : idx))}
                              sx={{ p: 0, mr: 1 }}
                            >
                              {expandedReasoningIdx === idx ? <ExpandLess /> : <ExpandMore />}
                            </IconButton>
                            <Typography
                              component="span"
                              variant="caption"
                              color="text.secondary"
                              sx={{ cursor: 'pointer' }}
                              onClick={() => setExpandedReasoningIdx((prev: number | null) => (prev === idx ? null : idx))}
                            >
                              Reasoning
                            </Typography>
                            <Collapse in={expandedReasoningIdx === idx}>
                              <Typography
                                variant="body2"
                                color="text.secondary"
                                sx={{
                                  mt: 1,
                                  fontStyle: 'italic',
                                  whiteSpace: 'pre-wrap',
                                }}
                              >
                                {msg.reasoning ?? reasoning}
                              </Typography>
                            </Collapse>
                            {(msg.content ?? response) && <Box sx={{ mt: 1 }} />}
                          </>
                        )}
                        {msg.streaming && progressSteps.length > 0 && (
                          <Box sx={{ mb: 1 }}>
                            <Typography variant="caption" color="text.secondary" display="block" sx={{ mb: 0.5 }}>
                              Progress
                            </Typography>
                            <Stack component="ul" sx={{ m: 0, pl: 2.5, listStyle: 'none' }}>
                              {progressSteps.map((step: ProgressStep) => (
                                <Box
                                  component="li"
                                  key={step.id}
                                  sx={{
                                    display: 'flex',
                                    alignItems: 'flex-start',
                                    gap: 0.5,
                                    py: 0.25,
                                    fontSize: '0.8125rem',
                                    color: step.status === 'done' ? 'text.secondary' : 'text.primary',
                                  }}
                                >
                                  <Typography component="span" variant="body2" color="inherit">
                                    {step.status === 'running' ? '⟳' : '✓'} {step.toolName}
                                  </Typography>
                                  {step.description && (
                                    <Typography component="span" variant="caption" color="text.secondary" sx={{ flex: 1 }}>
                                      — {step.description.length > 60 ? `${step.description.slice(0, 60)}…` : step.description}
                                    </Typography>
                                  )}
                                </Box>
                              ))}
                            </Stack>
                          </Box>
                        )}
                        {!msg.streaming && (msg.progressSteps?.length ?? 0) > 0 && (
                          <Box sx={{ mb: 1 }}>
                            <IconButton
                              size="small"
                              onClick={() => setExpandedProgressIdx((prev: number | null) => (prev === idx ? null : idx))}
                              sx={{ p: 0, mr: 1 }}
                            >
                              {expandedProgressIdx === idx ? <ExpandLess /> : <ExpandMore />}
                            </IconButton>
                            <Typography
                              component="span"
                              variant="caption"
                              color="text.secondary"
                              sx={{ cursor: 'pointer' }}
                              onClick={() => setExpandedProgressIdx((prev: number | null) => (prev === idx ? null : idx))}
                            >
                              Progress
                            </Typography>
                            <Collapse in={expandedProgressIdx === idx}>
                              <Stack component="ul" sx={{ m: 0, pl: 2.5, listStyle: 'none', mt: 0.5 }}>
                                {(msg.progressSteps ?? []).map((step: ProgressStep) => (
                                  <Box
                                    component="li"
                                    key={step.id}
                                    sx={{
                                      display: 'flex',
                                      alignItems: 'flex-start',
                                      gap: 0.5,
                                      py: 0.25,
                                      fontSize: '0.8125rem',
                                      color: 'text.secondary',
                                    }}
                                  >
                                    <Typography component="span" variant="body2" color="inherit">
                                      ✓ {step.toolName}
                                    </Typography>
                                    {step.description && (
                                      <Typography component="span" variant="caption" color="text.secondary" sx={{ flex: 1 }}>
                                        — {step.description.length > 60 ? `${step.description.slice(0, 60)}…` : step.description}
                                      </Typography>
                                    )}
                                  </Box>
                                ))}
                              </Stack>
                            </Collapse>
                          </Box>
                        )}
                        {(msg.content || response || '').trim() ? (
                          <Markdown
                            remarkPlugins={[remarkGfm]}
                            rehypePlugins={[rehypeRaw]}
                            components={{
                              code: ({ children }: { children?: ReactNode }) => (
                                <CodeBlock>{children}</CodeBlock>
                              ),
                              img: ({ src, alt }: { src?: string; alt?: string }) => {
                                if (src && isGrafanaRenderImageSrc(src)) {
                                  const dashboardHref = dashboardUrlFromRenderUrl(src);
                                  return (
                                    <GrafanaPanelImage
                                      src={src}
                                      alt={alt ?? 'Grafana panel'}
                                      dashboardHref={dashboardHref}
                                    />
                                  );
                                }
                                return <Box component="img" src={src} alt={alt ?? ''} />;
                              },
                            }}
                          >
                            {msg.content || response}
                          </Markdown>
                        ) : msg.streaming && loading && !response ? (
                          <Typography color="text.secondary" variant="body2">
                            {progressSteps.length > 0 ? 'Working…' : 'Typing...'}
                          </Typography>
                        ) : null}
                      </Box>
                    )}
                  </Box>
                </Box>
              ))}
              </Box>
            )}
            <div ref={messagesEndRef} />
          </Box>
          <Stack>
            <TextField
              size="small"
              placeholder="Message ADRE..."
              value={ask}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => setAsk(e.target.value)}
              onKeyDown={(e: React.KeyboardEvent) => e.key === 'Enter' && !e.shiftKey && handleSend()}
              fullWidth
              multiline
              minRows={2}
              maxRows={6}
              sx={{
                '& .MuiOutlinedInput-root': {
                  bgcolor: '#1e1e1e',
                  '& fieldset': { borderColor: 'rgba(255,255,255,0.12)' },
                },
              }}
            />
            <Stack direction="row" justifyContent="flex-end" sx={{ mt: 0.5 }}>
              <IconButton
                size="small"
                onClick={handleSend}
                disabled={loading || !ask.trim()}
                sx={{ color: 'primary.main' }}
                aria-label="Send"
              >
                <Send />
              </IconButton>
            </Stack>
          </Stack>
        </Stack>
    </Box>
  );
};
