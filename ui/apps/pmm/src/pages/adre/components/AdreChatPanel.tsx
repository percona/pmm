import {
  Box,
  Button,
  Card,
  CardContent,
  Collapse,
  FormControl,
  IconButton,
  InputLabel,
  MenuItem,
  Select,
  Stack,
  ToggleButton,
  ToggleButtonGroup,
  TextField,
  Typography,
} from '@mui/material';
import ExpandLess from '@mui/icons-material/ExpandLess';
import ExpandMore from '@mui/icons-material/ExpandMore';
import { FC, useState, useCallback, useEffect, useRef, ReactNode } from 'react';
import Markdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import rehypeRaw from 'rehype-raw';
import { useAdreModels, useAdreSettings } from 'hooks/api/useAdre';
import { adreChatStream } from 'api/adre';
import { useSnackbar } from 'notistack';
import { CodeBlock } from 'pages/updates/change-log/code-block';

const STORAGE_KEY = 'pmm-adre-chat';
const CHAT_HISTORY_WINDOW_MS = 24 * 60 * 60 * 1000;
const CHAT_HISTORY_MAX_MESSAGES = 300;

interface ChatMessage {
  role: 'user' | 'assistant';
  content: string;
  timestamp?: number;
  reasoning?: string;
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
        ? (parsed.history as ChatMessage[]).filter(
            (m): m is ChatMessage =>
              m && typeof m === 'object' && (m.role === 'user' || m.role === 'assistant') && typeof m.content === 'string'
          )
        : [];
      const history = getWindowedHistory(rawHistory);
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
  const [history, setHistory] = useState<ChatMessage[]>(() => loadFromStorage().history);
  const [expandedReasoningIdx, setExpandedReasoningIdx] = useState<number | null>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const streamStartTimeRef = useRef<number | null>(null);

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

  const handleSend = useCallback(async () => {
    if (!ask.trim()) return;
    const userAsk = ask.trim();
    setLoading(true);
    setResponse('');
    setReasoning('');
    streamStartTimeRef.current = Date.now();
    setAsk('');
    const userTimestamp = Date.now();
    setHistory((prev: ChatMessage[]) => [...prev, { role: 'user', content: userAsk, timestamp: userTimestamp }]);
    try {
      const windowed = getWindowedHistory(history);
      const req = {
        ask: userAsk,
        conversationHistory: [
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
      await adreChatStream(req, (contentChunk, reasoningChunk) => {
        if (contentChunk) fullResponse += contentChunk;
        if (reasoningChunk) fullReasoning += reasoningChunk;
        setReasoning(fullReasoning);
        setResponse(fullResponse);
      });
      setHistory((prev: ChatMessage[]) => [
        ...prev,
        {
          role: 'assistant',
          content: fullResponse,
          timestamp: Date.now(),
          reasoning: fullReasoning || undefined,
        },
      ]);
      setResponse('');
      setReasoning('');
    } catch (err) {
      enqueueSnackbar(err instanceof Error ? err.message : 'Chat request failed', { variant: 'error' });
    } finally {
      setLoading(false);
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
    <Card variant="outlined" sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <CardContent sx={{ flex: 1, display: 'flex', flexDirection: 'column', minHeight: 0 }}>
        <Stack direction="row" alignItems="center" justifyContent="space-between" flexWrap="wrap" gap={1} sx={{ mb: 1 }}>
          <Typography variant="h6">Chat</Typography>
          <ToggleButtonGroup
            value={mode}
            exclusive
            onChange={(_, v) => v != null && setMode(v)}
            size="small"
            sx={{ '& .MuiToggleButton-root': { py: 0.25, px: 1 } }}
          >
            <ToggleButton value="chat" aria-label="Chat (fast)">
              Chat (fast)
            </ToggleButton>
            <ToggleButton value="investigation" aria-label="Investigation">
              Investigation
            </ToggleButton>
          </ToggleButtonGroup>
        </Stack>
        <Stack gap={1} sx={{ flex: 1, minHeight: 0 }}>
          <Box
            ref={containerRef}
            id="messages-container"
            sx={{
              flex: 1,
              minHeight: 160,
              maxHeight: 450,
              overflow: 'auto',
              p: 2,
              display: 'flex',
              flexDirection: 'column',
              gap: 2,
              bgcolor: 'action.hover',
              borderRadius: 1,
            }}
          >
            {allMessages.length === 0 ? (
              <Typography color="text.secondary" variant="body2" sx={{ alignSelf: 'center', mt: 2 }}>
                Ask a question about your database environment...
              </Typography>
            ) : (
              allMessages.map((msg, idx) => (
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
                            bgcolor: 'primary.main',
                            color: 'primary.contrastText',
                          }
                        : {
                            bgcolor: 'background.paper',
                            border: 1,
                            borderColor: 'divider',
                            boxShadow: 1,
                          }),
                    }}
                  >
                    <Typography
                      variant="caption"
                      color={msg.role === 'user' ? undefined : 'text.secondary'}
                      display="block"
                      sx={{ mb: 0.5, ...(msg.role === 'user' && { opacity: 0.9 }) }}
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
                        {(msg.content || response || '').trim() ? (
                          <Markdown
                            remarkPlugins={[remarkGfm]}
                            rehypePlugins={[rehypeRaw]}
                            components={{
                              code: ({ children }: { children?: ReactNode }) => (
                                <CodeBlock>{children}</CodeBlock>
                              ),
                            }}
                          >
                            {msg.content || response}
                          </Markdown>
                        ) : msg.streaming && loading && !response ? (
                          <Typography color="text.secondary" variant="body2">
                            Typing...
                          </Typography>
                        ) : null}
                      </Box>
                    )}
                  </Box>
                </Box>
              ))
            )}
            <div ref={messagesEndRef} />
          </Box>
          <Stack direction="row" gap={1} alignItems="center">
            <FormControl size="small" sx={{ minWidth: 140 }}>
              <InputLabel>Model</InputLabel>
              <Select value={model} label="Model" onChange={(e: { target: { value: string } }) => setModel(e.target.value)}>
                <MenuItem value="">Default</MenuItem>
                {models.map((m: string) => (
                  <MenuItem key={m} value={m}>
                    {m}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
          </Stack>
          <Stack direction="row" gap={1}>
            <TextField
              size="small"
              placeholder="Ask something..."
              value={ask}
                onChange={(e: React.ChangeEvent<HTMLInputElement>) => setAsk(e.target.value)}
              onKeyDown={(e: React.KeyboardEvent) => e.key === 'Enter' && !e.shiftKey && handleSend()}
              fullWidth
              multiline
              minRows={2}
              maxRows={6}
            />
            <Button variant="contained" onClick={handleSend} disabled={loading || !ask.trim()}>
              Send
            </Button>
          </Stack>
        </Stack>
      </CardContent>
    </Card>
  );
};
