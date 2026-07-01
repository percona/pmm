import {
  Alert,
  Box,
  IconButton,
  Stack,
  TextField,
  Typography,
} from '@mui/material';
import AutoAwesomeIcon from '@mui/icons-material/AutoAwesome';
import KeyboardArrowUpIcon from '@mui/icons-material/KeyboardArrowUp';
import { FC, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import Markdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import rehypeRaw from 'rehype-raw';
import { useAdreChat, formatTimestamp } from 'hooks/useAdreChat';
import { useAdreSettings } from 'hooks/api/useAdre';
import { getMarkdownComponents } from 'components/adre/adre-chat-markdown.helpers';
import { HolmesUsageFooter } from 'components/adre/HolmesUsageFooter';
import { useQanPanelState } from '../hooks/useQanPanelState';
import { useQanServiceId } from '../hooks/useQanServiceId';
import { getLabelQueryParams } from '../utils/qanTools';

const ADVISORY_FOOTER =
  'Recommendations are advisory. Copy SQL and apply manually — PMM does not execute changes.';

export const QanAiAside: FC = () => {
  const state = useQanPanelState();
  const serviceId = useQanServiceId();
  const { data: settings } = useAdreSettings();
  const {
    response,
    reasoning,
    loading,
    progressSteps,
    allMessages,
    chatError,
    handleSend,
    resetEphemeralChat,
  } = useAdreChat({ context: 'qan-aside' });
  const [ask, setAsk] = useState('');
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const lastQueryKeyRef = useRef<string>('');

  const queryKey = `${state.queryId ?? ''}:${serviceId}`;

  useEffect(() => {
    if (queryKey === lastQueryKeyRef.current) return;
    lastQueryKeyRef.current = queryKey;
    resetEphemeralChat();
    setAsk('');
  }, [queryKey, resetEphemeralChat]);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [allMessages.length, response, loading]);

  const qanContext = useMemo(() => {
    const payload = {
      type: 'native_qan',
      serviceId,
      queryId: state.queryId,
      fingerprint: state.fingerprint,
      tab: state.openDetailsTab,
      groupBy: state.groupBy,
      from: state.from,
      to: state.to,
      labels: getLabelQueryParams(state.labels),
    };
    return `Native QAN context (advisory only — do not apply changes automatically):\n${JSON.stringify(payload, null, 2)}`;
  }, [state, serviceId]);

  const onSend = useCallback(() => {
    const text = ask.trim();
    if (!text || loading) return;
    setAsk('');
    handleSend(text, { dashboardContext: qanContext, mode: 'fast' });
  }, [ask, loading, handleSend, qanContext]);

  const configured = settings?.enabled && settings?.url;

  return (
    <Stack
      sx={{
        width: 400,
        flexShrink: 0,
        borderLeft: 1,
        borderColor: 'divider',
        minHeight: 0,
        bgcolor: 'background.paper',
      }}
      data-testid="qan-ai-aside"
    >
      <Box sx={{ px: 2, py: 1.5, borderBottom: 1, borderColor: 'divider' }}>
        <Stack direction="row" alignItems="center" spacing={1}>
          <AutoAwesomeIcon color="primary" fontSize="small" />
          <Box>
            <Typography variant="subtitle1" sx={{ fontWeight: 600, lineHeight: 1.2 }}>
              AI Assistant
            </Typography>
            <Typography variant="caption" color="text.secondary">
              Query context · advisory only
            </Typography>
          </Box>
        </Stack>
      </Box>
      {!configured ? (
        <Alert severity="info" sx={{ m: 2 }}>
          Configure ADRE in Settings → AI Assistant to use chat.
        </Alert>
      ) : (
        <>
          <Box sx={{ flex: 1, overflow: 'auto', px: 2, py: 1.5 }}>
            {allMessages.map((msg, idx) => (
              <Box
                key={msg.serverMessageId ?? idx}
                sx={{
                  display: 'flex',
                  justifyContent: msg.role === 'user' ? 'flex-end' : 'flex-start',
                  mb: 1.5,
                  maxWidth: '100%',
                }}
              >
                <Box
                  sx={{
                    px: 1.5,
                    py: 1,
                    borderRadius: 2,
                    maxWidth: '95%',
                    ...(msg.role === 'user'
                      ? { bgcolor: 'action.selected' }
                      : {
                          bgcolor: 'rgba(255,255,255,0.05)',
                          border: 1,
                          borderColor: 'divider',
                        }),
                    fontSize: '0.85rem',
                  }}
                >
                  <Typography variant="caption" color="text.secondary" display="block" sx={{ mb: 0.25 }}>
                    {msg.role === 'user' ? 'You' : 'Assistant'}
                    {msg.timestamp != null ? ` · ${formatTimestamp(msg.timestamp)}` : ''}
                  </Typography>
                  <Markdown
                    remarkPlugins={[remarkGfm]}
                    rehypePlugins={[rehypeRaw]}
                    components={getMarkdownComponents(msg.content || '')}
                  >
                    {msg.content || ''}
                  </Markdown>
                  {msg.usage ? <HolmesUsageFooter usage={msg.usage} /> : null}
                </Box>
              </Box>
            ))}
            {loading && response ? (
              <Box
                sx={{
                  px: 1.5,
                  py: 1,
                  borderRadius: 2,
                  border: 1,
                  borderColor: 'divider',
                  bgcolor: 'rgba(255,255,255,0.05)',
                  fontSize: '0.85rem',
                }}
              >
                <Markdown
                  remarkPlugins={[remarkGfm]}
                  rehypePlugins={[rehypeRaw]}
                  components={getMarkdownComponents(response)}
                >
                  {response}
                </Markdown>
              </Box>
            ) : null}
            {reasoning ? (
              <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 1 }}>
                Reasoning: {reasoning.slice(0, 200)}…
              </Typography>
            ) : null}
            {progressSteps.length ? (
              <Typography variant="caption" color="text.secondary">
                {progressSteps.map((s) => s.description ?? s.toolName).join(' · ')}
              </Typography>
            ) : null}
            {chatError ? <Alert severity="error" sx={{ mt: 1 }}>{chatError}</Alert> : null}
            <div ref={messagesEndRef} />
          </Box>
          <Stack sx={{ p: 2, borderTop: 1, borderColor: 'divider' }} spacing={1}>
            <Stack direction="row" alignItems="flex-end" spacing={0.5}>
              <TextField
                multiline
                minRows={2}
                maxRows={4}
                size="small"
                fullWidth
                placeholder="Ask about queries, metrics, performance…"
                value={ask}
                onChange={(e) => setAsk(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && !e.shiftKey) {
                    e.preventDefault();
                    onSend();
                  }
                }}
                disabled={loading}
                sx={{
                  '& .MuiOutlinedInput-root': { borderRadius: 2 },
                }}
              />
              <IconButton
                color="primary"
                onClick={onSend}
                disabled={loading || !ask.trim()}
                aria-label="Send message"
                sx={{
                  bgcolor: 'action.selected',
                  mb: 0.25,
                  '&:hover': { bgcolor: 'action.focus' },
                }}
              >
                <KeyboardArrowUpIcon />
              </IconButton>
            </Stack>
            <Typography variant="caption" color="text.secondary">
              {ADVISORY_FOOTER}
            </Typography>
          </Stack>
        </>
      )}
    </Stack>
  );
};
