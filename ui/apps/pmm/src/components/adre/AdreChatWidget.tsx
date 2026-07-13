import {
  Box,
  Collapse,
  IconButton,
  Paper,
  Stack,
  TextField,
  Typography,
} from '@mui/material';
import ChatIcon from '@mui/icons-material/Chat';
import CloseIcon from '@mui/icons-material/Close';
import DeleteSweepIcon from '@mui/icons-material/DeleteSweep';
import SendIcon from '@mui/icons-material/Send';
import ExpandLess from '@mui/icons-material/ExpandLess';
import ExpandMore from '@mui/icons-material/ExpandMore';
import ContentCopy from '@mui/icons-material/ContentCopy';
import Check from '@mui/icons-material/Check';
import { FC, useState, useCallback, useEffect, useRef, useMemo, memo } from 'react';
import Markdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import rehypeRaw from 'rehype-raw';
import { useLocation } from 'react-router-dom';
import { useSnackbar } from 'notistack';
import { useAdreChat, formatTimestamp, type ProgressStep, type ChatMessage } from 'hooks/useAdreChat';
import { useGrafana } from 'contexts/grafana';
import { PanelScrollRootProvider } from 'components/adre/adre-chat-markdown';
import { getMarkdownComponents } from 'components/adre/adre-chat-markdown.helpers';
import { HolmesUsageFooter } from 'components/adre/HolmesUsageFooter';
import { buildGrafanaDashboardContext } from 'components/adre/grafana-context';

interface ChatMessageBubbleProps {
  msg: ChatMessage & { streaming?: boolean };
  idx: number;
  expandedReasoningIdx: number | null;
  setExpandedReasoningIdx: React.Dispatch<React.SetStateAction<number | null>>;
  expandedProgressIdx: number | null;
  setExpandedProgressIdx: React.Dispatch<React.SetStateAction<number | null>>;
  reasoning: string;
  response: string;
  loading: boolean;
  progressSteps: ProgressStep[];
  copiedMessageKey: string | null;
  onCopyMessage: (key: string, text: string) => void;
}

const ChatMessageBubble = memo<ChatMessageBubbleProps>(({
  msg, idx, expandedReasoningIdx, setExpandedReasoningIdx,
  expandedProgressIdx, setExpandedProgressIdx,
  reasoning, response, loading, progressSteps, copiedMessageKey, onCopyMessage,
}) => {
  const mdComponents = useMemo(
    () => getMarkdownComponents(msg.content || response || ''),
    [msg.content, response]
  );
  const messageKey = String(msg.serverMessageId ?? `row-${idx}`);
  const messageText = (msg.content || response || '').trim();

  return (
    <Box
      sx={{
        display: 'flex',
        justifyContent: msg.role === 'user' ? 'flex-end' : 'flex-start',
        alignSelf: msg.role === 'user' ? 'flex-end' : 'flex-start',
        maxWidth: '90%',
      }}
    >
      <Box
        sx={{
          px: 1.5,
          py: 1,
          borderRadius: 2,
          ...(msg.role === 'user'
            ? { bgcolor: '#2d3748', color: 'text.primary' }
            : { bgcolor: 'rgba(255,255,255,0.05)', border: 1, borderColor: 'rgba(255,255,255,0.12)' }),
          '& img': { maxWidth: '100%', height: 'auto', display: 'block' },
          '& pre': { maxWidth: '100%', minWidth: 0 },
          fontSize: '0.85rem',
        }}
      >
        <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 0.25 }}>
          <Typography
            variant="caption"
            color="text.secondary"
            display="block"
            sx={{ fontSize: '0.65rem', opacity: 0.8 }}
          >
            {msg.role === 'user' ? 'You' : 'Assistant'}
            {msg.timestamp ? ` · ${formatTimestamp(msg.timestamp)}` : ''}
          </Typography>
          {msg.role === 'assistant' && messageText ? (
            <IconButton
              size="small"
              title={copiedMessageKey === messageKey ? 'Copied' : 'Copy response'}
              onClick={() => onCopyMessage(messageKey, messageText)}
              sx={{ p: 0.25, color: copiedMessageKey === messageKey ? 'success.light' : 'text.secondary' }}
            >
              {copiedMessageKey === messageKey ? <Check fontSize="inherit" /> : <ContentCopy fontSize="inherit" />}
            </IconButton>
          ) : null}
        </Stack>
        {msg.role === 'user' ? (
          <Typography variant="body2" sx={{ whiteSpace: 'pre-wrap' }}>{msg.content}</Typography>
        ) : (
          <Box>
            {(msg.reasoning ?? (msg.streaming && reasoning)) && (
              <>
                <IconButton
                  size="small"
                  onClick={() => setExpandedReasoningIdx((prev) => (prev === idx ? null : idx))}
                  sx={{ p: 0, mr: 0.5 }}
                >
                  {expandedReasoningIdx === idx ? <ExpandLess fontSize="small" /> : <ExpandMore fontSize="small" />}
                </IconButton>
                <Typography
                  component="span"
                  variant="caption"
                  color="text.secondary"
                  sx={{ cursor: 'pointer', fontSize: '0.75rem' }}
                  onClick={() => setExpandedReasoningIdx((prev) => (prev === idx ? null : idx))}
                >
                  Reasoning
                </Typography>
                <Collapse in={expandedReasoningIdx === idx}>
                  <Typography
                    variant="body2"
                    color="text.secondary"
                    sx={{ mt: 0.5, fontStyle: 'italic', whiteSpace: 'pre-wrap', fontSize: '0.8rem' }}
                  >
                    {msg.reasoning ?? reasoning}
                  </Typography>
                </Collapse>
                {(msg.content ?? response) && <Box sx={{ mt: 0.5 }} />}
              </>
            )}
            {msg.streaming && progressSteps.length > 0 && (
              <Box sx={{ mb: 0.5 }}>
                <Typography variant="caption" color="text.secondary" display="block" sx={{ mb: 0.25, fontSize: '0.7rem' }}>
                  Progress
                </Typography>
                <Stack component="ul" sx={{ m: 0, pl: 2, listStyle: 'none' }}>
                  {progressSteps.map((step: ProgressStep) => (
                    <Box
                      component="li"
                      key={step.id}
                      sx={{
                        display: 'flex',
                        alignItems: 'flex-start',
                        gap: 0.5,
                        py: 0.15,
                        fontSize: '0.75rem',
                        color: step.status === 'done' ? 'text.secondary' : 'text.primary',
                      }}
                    >
                      <Typography component="span" variant="caption" color="inherit">
                        {step.status === 'running' ? '⟳' : '✓'} {step.toolName}
                      </Typography>
                    </Box>
                  ))}
                </Stack>
              </Box>
            )}
            {!msg.streaming && (msg.progressSteps?.length ?? 0) > 0 && (
              <Box sx={{ mb: 0.5 }}>
                <IconButton
                  size="small"
                  onClick={() => setExpandedProgressIdx((prev) => (prev === idx ? null : idx))}
                  sx={{ p: 0, mr: 0.5 }}
                >
                  {expandedProgressIdx === idx ? <ExpandLess fontSize="small" /> : <ExpandMore fontSize="small" />}
                </IconButton>
                <Typography
                  component="span"
                  variant="caption"
                  color="text.secondary"
                  sx={{ cursor: 'pointer', fontSize: '0.7rem' }}
                  onClick={() => setExpandedProgressIdx((prev) => (prev === idx ? null : idx))}
                >
                  Progress
                </Typography>
                <Collapse in={expandedProgressIdx === idx}>
                  <Stack component="ul" sx={{ m: 0, pl: 2, listStyle: 'none', mt: 0.25 }}>
                    {(msg.progressSteps ?? []).map((step: ProgressStep) => (
                      <Box
                        component="li"
                        key={step.id}
                        sx={{
                          display: 'flex',
                          alignItems: 'flex-start',
                          gap: 0.5,
                          py: 0.15,
                          fontSize: '0.75rem',
                          color: 'text.secondary',
                        }}
                      >
                        <Typography component="span" variant="caption" color="inherit">
                          ✓ {step.toolName}
                        </Typography>
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
                components={mdComponents}
              >
                {msg.content || response}
              </Markdown>
            ) : msg.streaming && loading && !response ? (
              <Typography color="text.secondary" variant="body2" sx={{ fontSize: '0.8rem' }}>
                {progressSteps.length > 0 ? 'Working…' : 'Typing...'}
              </Typography>
            ) : null}
            {!msg.streaming && msg.role === 'assistant' && msg.usage ? (
              <HolmesUsageFooter usage={msg.usage} />
            ) : null}
          </Box>
        )}
      </Box>
    </Box>
  );
});

export const AdreChatWidget: FC = () => {
  const { loading, progressSteps, allMessages, settings, response, reasoning, handleSend, clearHistory } = useAdreChat();
  const { enqueueSnackbar } = useSnackbar();
  const location = useLocation();
  const { grafanaDocumentTitle } = useGrafana();
  const [open, setOpen] = useState(false);
  const [ask, setAsk] = useState('');
  const [expandedReasoningIdx, setExpandedReasoningIdx] = useState<number | null>(null);
  const [expandedProgressIdx, setExpandedProgressIdx] = useState<number | null>(null);
  const [copiedMessageKey, setCopiedMessageKey] = useState<string | null>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const [scrollRoot, setScrollRoot] = useState<HTMLElement | null>(null);
  const lastScrollRef = useRef(0);
  const copyResetTimerRef = useRef<number | null>(null);

  const isConfigured = settings?.enabled && !!settings?.url;
  const chatViaLabel = isConfigured ? 'ADRE chat' : 'ADRE';
  const hideOnNativeQan =
    location.pathname.includes('/qan') && !location.pathname.includes('/qan/ai-insights');

  const scrollToBottom = useCallback((instant?: boolean) => {
    const now = Date.now();
    if (!instant && now - lastScrollRef.current < 200) return;
    lastScrollRef.current = now;
    messagesEndRef.current?.scrollIntoView({ behavior: instant ? 'auto' : 'smooth' });
  }, []);

  useEffect(() => {
    if (open) scrollToBottom(loading);
  }, [allMessages.length, response, reasoning, loading, open, scrollToBottom]);

  useEffect(() => {
    if (open) {
      const id = requestAnimationFrame(() => {
        messagesEndRef.current?.scrollIntoView({ behavior: 'auto' });
      });

      return () => cancelAnimationFrame(id);
    }
  }, [open]);

  useEffect(() => () => {
    if (copyResetTimerRef.current != null) {
      window.clearTimeout(copyResetTimerRef.current);
    }
  }, []);

  const onCopyMessage = useCallback(async (key: string, text: string) => {
    if (!text.trim()) return;
    try {
      await navigator.clipboard.writeText(text);
      setCopiedMessageKey(key);
      if (copyResetTimerRef.current != null) {
        window.clearTimeout(copyResetTimerRef.current);
      }
      copyResetTimerRef.current = window.setTimeout(() => {
        setCopiedMessageKey((prev: string | null) => (prev === key ? null : prev));
      }, 1600);
    } catch {
      setCopiedMessageKey(null);
    }
  }, []);

  const onSend = useCallback(async () => {
    if (!ask.trim() || !isConfigured) return;
    const userAsk = ask;
    setAsk('');
    const dashboardContext = buildGrafanaDashboardContext(
      location.pathname,
      location.search,
      window.location.origin,
      grafanaDocumentTitle,
    );
    await handleSend(userAsk, { dashboardContext: dashboardContext || undefined });
  }, [ask, isConfigured, location.pathname, location.search, handleSend, grafanaDocumentTitle]);

  if (!isConfigured || hideOnNativeQan) return null;

  return (
    <>
      <IconButton
        onClick={() => setOpen((o) => !o)}
        sx={{
          position: 'fixed',
          bottom: 24,
          right: 24,
          bgcolor: 'primary.main',
          color: 'primary.contrastText',
          width: 56,
          height: 56,
          boxShadow: 2,
          '&:hover': { bgcolor: 'primary.dark' },
          zIndex: 1300,
        }}
        aria-label="Open ADRE chat"
      >
        {open ? <CloseIcon /> : <ChatIcon />}
      </IconButton>
      {open && (
        <Paper
          elevation={8}
          sx={{
            position: 'fixed',
            bottom: 90,
            right: 24,
            width: 420,
            maxWidth: 'calc(100vw - 48px)',
            height: 520,
            display: 'flex',
            flexDirection: 'column',
            overflow: 'hidden',
            zIndex: 1300,
          }}
        >
          <Stack
            direction="row"
            alignItems="center"
            justifyContent="space-between"
            sx={{ p: 1, borderBottom: 1, borderColor: 'divider' }}
          >
            <Stack>
              <Typography variant="subtitle1">ADRE Chat</Typography>
              <Typography variant="caption" color="text.secondary">
                {chatViaLabel}
              </Typography>
            </Stack>
            <Stack direction="row" gap={0.5}>
              <IconButton
                size="small"
                onClick={() => {
                  clearHistory();
                  enqueueSnackbar('Conversation cleared', { variant: 'info', autoHideDuration: 2000 });
                }}
                title="New conversation"
              >
                <DeleteSweepIcon fontSize="small" />
              </IconButton>
              <IconButton size="small" onClick={() => setOpen(false)}>
                <CloseIcon />
              </IconButton>
            </Stack>
          </Stack>
          <Box
            ref={(el) => setScrollRoot(el as HTMLElement | null)}
            sx={{
              flex: 1,
              overflow: 'auto',
              p: 1,
              bgcolor: '#212121',
              display: 'flex',
              flexDirection: 'column',
              gap: 1,
            }}
          >
            <PanelScrollRootProvider value={scrollRoot}>
            {allMessages.length === 0 ? (
              <Typography color="text.secondary" variant="body2" sx={{ alignSelf: 'center', mt: 2 }}>
                Ask a question about your database environment...
              </Typography>
            ) : (
              allMessages.map((msg, idx) => (
                <ChatMessageBubble
                  key={`${msg.role}-${msg.timestamp ?? idx}`}
                  msg={msg}
                  idx={idx}
                  expandedReasoningIdx={expandedReasoningIdx}
                  setExpandedReasoningIdx={setExpandedReasoningIdx}
                  expandedProgressIdx={expandedProgressIdx}
                  setExpandedProgressIdx={setExpandedProgressIdx}
                  reasoning={reasoning}
                  response={response}
                  loading={loading}
                  progressSteps={progressSteps}
                  copiedMessageKey={copiedMessageKey}
                  onCopyMessage={onCopyMessage}
                />
              ))
            )}
            <div ref={messagesEndRef} />
            </PanelScrollRootProvider>
          </Box>
          <Stack direction="row" gap={0.5} sx={{ p: 1 }}>
            <TextField
              size="small"
              placeholder="Message ADRE..."
              value={ask}
              onChange={(e) => setAsk(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && !e.shiftKey && onSend()}
              fullWidth
              sx={{
                '& .MuiOutlinedInput-root': {
                  fontSize: '0.85rem',
                  bgcolor: '#1e1e1e',
                  '& fieldset': { borderColor: 'rgba(255,255,255,0.12)' },
                },
              }}
            />
            <IconButton
              color="primary"
              onClick={onSend}
              disabled={loading || !ask.trim()}
            >
              <SendIcon />
            </IconButton>
          </Stack>
        </Paper>
      )}
    </>
  );
};
