import {
  Box,
  Collapse,
  IconButton,
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
import { FC, useState, useCallback, useEffect, useRef } from 'react';
import Markdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import rehypeRaw from 'rehype-raw';
import { useAdreModels } from 'hooks/api/useAdre';
import { useAdreChat, formatTimestamp, type ProgressStep } from 'hooks/useAdreChat';
import { getMarkdownComponents } from 'components/adre/adre-chat-markdown';

export const AdreChatPanel: FC = () => {
  const { data: models = [] } = useAdreModels();
  const { response, reasoning, loading, progressSteps, allMessages, settings, handleSend } = useAdreChat();
  const [ask, setAsk] = useState('');
  const [model, setModel] = useState('');
  const [mode, setMode] = useState<'chat' | 'investigation'>('chat');
  const [expandedReasoningIdx, setExpandedReasoningIdx] = useState<number | null>(null);
  const [expandedProgressIdx, setExpandedProgressIdx] = useState<number | null>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  const defaultModeSyncedRef = useRef(false);
  useEffect(() => {
    if (!defaultModeSyncedRef.current && (settings?.defaultChatMode === 'investigation' || settings?.defaultChatMode === 'chat')) {
      defaultModeSyncedRef.current = true;
      setMode(settings.defaultChatMode);
    }
  }, [settings?.defaultChatMode]);

  const lastScrollRef = useRef(0);
  const scrollToBottom = useCallback((instant?: boolean) => {
    const now = Date.now();
    if (!instant && now - lastScrollRef.current < 200) return;
    lastScrollRef.current = now;
    messagesEndRef.current?.scrollIntoView({ behavior: instant ? 'auto' : 'smooth' });
  }, []);

  useEffect(() => {
    scrollToBottom(loading);
  }, [allMessages.length, response, reasoning, loading, scrollToBottom]);

  useEffect(() => {
    const id = requestAnimationFrame(() => {
      messagesEndRef.current?.scrollIntoView({ behavior: 'auto' });
    });
    return () => cancelAnimationFrame(id);
  }, []);

  const onSend = useCallback(async () => {
    if (!ask.trim()) return;
    const userAsk = ask;
    setAsk('');
    await handleSend(userAsk, { model: model || undefined, mode });
  }, [ask, model, mode, handleSend]);

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
                            components={getMarkdownComponents(msg.content || response || '')}
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
              onKeyDown={(e: React.KeyboardEvent) => e.key === 'Enter' && !e.shiftKey && onSend()}
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
                onClick={onSend}
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
