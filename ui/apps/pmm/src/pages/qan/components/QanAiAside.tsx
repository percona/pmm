import {
  Alert,
  Box,
  IconButton,
  Stack,
  TextField,
  Typography,
} from '@mui/material';
import SendIcon from '@mui/icons-material/Send';
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
        <Typography variant="subtitle1">AI Assistant</Typography>
        <Typography variant="caption" color="text.secondary">
          Query context · advisory only
        </Typography>
      </Box>
      {!configured ? (
        <Alert severity="info" sx={{ m: 2 }}>
          Configure ADRE in Settings → AI Assistant to use chat.
        </Alert>
      ) : (
        <>
          <Box sx={{ flex: 1, overflow: 'auto', px: 2, py: 1 }}>
            {allMessages.map((msg, idx) => (
              <Box
                key={msg.serverMessageId ?? idx}
                sx={{
                  mb: 1.5,
                  alignSelf: msg.role === 'user' ? 'flex-end' : 'flex-start',
                  maxWidth: '95%',
                }}
              >
                <Typography variant="caption" color="text.secondary">
                  {msg.role}
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
            ))}
            {loading && response ? (
              <Markdown
                remarkPlugins={[remarkGfm]}
                rehypePlugins={[rehypeRaw]}
                components={getMarkdownComponents(response)}
              >
                {response}
              </Markdown>
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
            {chatError ? <Alert severity="error">{chatError}</Alert> : null}
            <div ref={messagesEndRef} />
          </Box>
          <Stack sx={{ p: 2, borderTop: 1, borderColor: 'divider' }} spacing={1}>
            <TextField
              multiline
              minRows={2}
              maxRows={4}
              size="small"
              placeholder="Ask about this query…"
              value={ask}
              onChange={(e) => setAsk(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && !e.shiftKey) {
                  e.preventDefault();
                  onSend();
                }
              }}
              disabled={loading}
            />
            <Stack direction="row" justifyContent="space-between" alignItems="center">
              <Typography variant="caption" color="text.secondary">
                {ADVISORY_FOOTER}
              </Typography>
              <IconButton color="primary" onClick={onSend} disabled={loading || !ask.trim()}>
                <SendIcon />
              </IconButton>
            </Stack>
          </Stack>
        </>
      )}
    </Stack>
  );
};
