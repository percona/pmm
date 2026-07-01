import {
  Alert,
  Box,
  Button,
  ButtonGroup,
  ClickAwayListener,
  Collapse,
  Grow,
  MenuItem,
  MenuList,
  Paper,
  Popper,
  IconButton,
  Stack,
  TextField,
  ToggleButton,
  ToggleButtonGroup,
  Tooltip,
  Typography,
} from '@mui/material';
import HelpOutline from '@mui/icons-material/HelpOutline';
import ExpandLess from '@mui/icons-material/ExpandLess';
import ExpandMore from '@mui/icons-material/ExpandMore';
import ContentCopy from '@mui/icons-material/ContentCopy';
import Check from '@mui/icons-material/Check';
import Send from '@mui/icons-material/Send';
import ArrowDropDownIcon from '@mui/icons-material/ArrowDropDown';
import { FC, useState, useCallback, useEffect, useLayoutEffect, useRef } from 'react';
import Markdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import rehypeRaw from 'rehype-raw';
import { useAdreModels } from 'hooks/api/useAdre';
import { useAdreChat, formatTimestamp, type ProgressStep } from 'hooks/useAdreChat';
import { AdreConversationsSidebar } from './AdreConversationsSidebar';
import { getMarkdownComponents } from 'components/adre/adre-chat-markdown.helpers';
import { HolmesUsageFooter } from 'components/adre/HolmesUsageFooter';
import {
  loadAdreChatUiPreferences,
  saveAdreChatUiPreferences,
  defaultChatModeFromSettings,
} from 'utils/adreChatUiPreferences';

export const AdreChatPanel: FC = () => {
  const { data: models = [], status: modelsQueryStatus } = useAdreModels();
  const {
    response,
    reasoning,
    loading,
    progressSteps,
    allMessages,
    settings,
    chatError,
    handleSend,
    conversationId,
    conversations,
    conversationsLoading,
    newChat,
    deleteConversation,
    selectConversation,
    searchHits,
    searchLoading,
    runSearch,
    scrollToMessageId,
    clearScrollToMessage,
  } = useAdreChat();
  const [ask, setAsk] = useState('');
  const [model, setModel] = useState('');
  const [mode, setMode] = useState<'fast' | 'investigation'>(() => {
    const p = loadAdreChatUiPreferences();
    if (p.mode === 'fast' || p.mode === 'investigation') return p.mode;
    return 'investigation';
  });
  const [modelMenuOpen, setModelMenuOpen] = useState(false);
  const [expandedReasoningIdx, setExpandedReasoningIdx] = useState<number | null>(null);
  const [expandedProgressIdx, setExpandedProgressIdx] = useState<number | null>(null);
  const [copiedMessageKey, setCopiedMessageKey] = useState<string | null>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const modelAnchorRef = useRef<HTMLDivElement>(null);
  const copyResetTimerRef = useRef<number | null>(null);
  /** After scrolling to a search hit, skip one "pin to bottom" so we do not jump away from the hit. */
  const skipPinToBottomOnceRef = useRef(false);

  const skipServerDefaultModeRef = useRef(
    (() => {
      const p = loadAdreChatUiPreferences();
      return p.mode === 'fast' || p.mode === 'investigation';
    })()
  );
  useEffect(() => {
    if (skipServerDefaultModeRef.current) return;
    if (settings === undefined) return;
    const dm = settings.defaultChatMode ?? settings.default_chat_mode;
    setMode(defaultChatModeFromSettings(typeof dm === 'string' ? dm : undefined));
  }, [settings]);
  const modelHydratedRef = useRef(false);
  useEffect(() => {
    if (modelHydratedRef.current || modelsQueryStatus !== 'success') return;
    modelHydratedRef.current = true;
    const p = loadAdreChatUiPreferences();
    if (p.model && models.includes(p.model)) {
      setModel(p.model);
    } else if (p.model) {
      saveAdreChatUiPreferences({ removeModel: true });
    }
  }, [models, modelsQueryStatus]);

  const setModePersist = useCallback((value: 'fast' | 'investigation') => {
    setMode(value);
    saveAdreChatUiPreferences({ mode: value });
  }, []);

  const setModelPersist = useCallback((value: string) => {
    setModel(value);
    saveAdreChatUiPreferences({ model: value });
  }, []);

  /** Keep view pinned to latest messages: instant jump (no smooth scroll through history on load/refresh). */
  useLayoutEffect(() => {
    if (scrollToMessageId != null) {
      return;
    }
    if (skipPinToBottomOnceRef.current) {
      skipPinToBottomOnceRef.current = false;
      return;
    }
    const el = containerRef.current;
    if (!el || allMessages.length === 0) {
      return;
    }
    el.scrollTop = el.scrollHeight;
  }, [
    conversationId,
    allMessages.length,
    response,
    reasoning,
    loading,
    scrollToMessageId,
  ]);

  useLayoutEffect(() => {
    if (scrollToMessageId == null || !containerRef.current) {
      return;
    }
    const root = containerRef.current;
    const el = root.querySelector(`[data-adre-msg-id="${scrollToMessageId}"]`);
    if (el instanceof HTMLElement) {
      el.scrollIntoView({ behavior: 'auto', block: 'center' });
      skipPinToBottomOnceRef.current = true;
    }
    clearScrollToMessage();
  }, [scrollToMessageId, allMessages.length, clearScrollToMessage]);

  const onSend = useCallback(async () => {
    if (!ask.trim()) return;
    const userAsk = ask;
    setAsk('');
    await handleSend(userAsk, { model: model || undefined, mode });
  }, [ask, model, mode, handleSend]);

  useEffect(() => () => {
    if (copyResetTimerRef.current != null) {
      window.clearTimeout(copyResetTimerRef.current);
    }
  }, []);

  const copyAssistantMessage = useCallback(async (key: string, text: string) => {
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

  const selectedModelLabel = model || 'Default';

  return (
    <Box
      sx={{
        flex: 1,
        minHeight: 0,
        width: '100%',
        maxWidth: '100%',
        minWidth: 0,
        display: 'flex',
        flexDirection: { xs: 'column', md: 'row' },
        gap: { xs: 1, md: 0 },
        overflow: 'hidden',
      }}
    >
      <Box
        sx={{
          flex: {
            xs: '0 0 auto',
            md: '0 1 clamp(200px, 22vw, 260px)',
          },
          width: { md: 'clamp(200px, 22vw, 260px)' },
          maxWidth: { xs: '100%', md: '260px' },
          minWidth: 0,
          maxHeight: { xs: 'min(36vh, 220px)', md: 'none' },
          minHeight: 0,
          display: 'flex',
          flexDirection: 'column',
          overflow: 'hidden',
        }}
      >
        <AdreConversationsSidebar
          conversationId={conversationId}
          conversations={conversations}
          loading={conversationsLoading}
          searchHits={searchHits}
          searchLoading={searchLoading}
          onNewChat={newChat}
          onDeleteConversation={deleteConversation}
          onSelectConversation={selectConversation}
          onSearch={runSearch}
        />
      </Box>
      <Stack gap={1} sx={{ flex: 1, minHeight: 0, minWidth: 0, overflow: 'hidden' }}>
          {chatError ? <Alert severity="error">{chatError}</Alert> : null}
          <Box
            ref={containerRef}
            id="messages-container"
            sx={{
              flex: 1,
              minHeight: 0,
              minWidth: 0,
              overflowY: 'auto',
              overflowX: 'hidden',
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
              <Box
                sx={{
                  maxWidth: '100%',
                  width: '100%',
                  minWidth: 0,
                  alignSelf: 'stretch',
                  display: 'flex',
                  flexDirection: 'column',
                  gap: 2,
                }}
              >
              {allMessages.map((msg, idx) => {
                const messageKey = String(msg.serverMessageId ?? `row-${idx}`);
                const messageText = (msg.content || response || '').trim();
                return (
                <Box
                  key={messageKey}
                  data-adre-msg-id={msg.serverMessageId != null ? String(msg.serverMessageId) : undefined}
                  sx={{
                    display: 'flex',
                    justifyContent: msg.role === 'user' ? 'flex-end' : 'flex-start',
                    alignSelf: msg.role === 'user' ? 'flex-end' : 'flex-start',
                    maxWidth: '100%',
                    minWidth: 0,
                  }}
                >
                  <Box
                    sx={{
                      maxWidth: { xs: '92%', sm: '88%', md: '85%' },
                      minWidth: 0,
                      px: 2,
                      py: 1.5,
                      borderRadius: 2,
                      '& img': { maxWidth: '100%', height: 'auto', display: 'block' },
                      '& pre': { maxWidth: '100%', minWidth: 0 },
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
                    <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 0.5 }}>
                      <Typography
                        variant="caption"
                        color={msg.role === 'user' ? 'text.secondary' : 'text.secondary'}
                        display="block"
                        sx={{ fontSize: '0.7rem', opacity: 0.8 }}
                      >
                        {msg.role === 'user' ? 'You' : 'Assistant'}
                        {msg.timestamp ? ` · ${formatTimestamp(msg.timestamp)}` : ''}
                      </Typography>
                      {msg.role === 'assistant' && messageText ? (
                        <IconButton
                          size="small"
                          title={copiedMessageKey === messageKey ? 'Copied' : 'Copy response'}
                          onClick={() => copyAssistantMessage(messageKey, messageText)}
                          sx={{ p: 0.25, color: copiedMessageKey === messageKey ? 'success.light' : 'text.secondary' }}
                        >
                          {copiedMessageKey === messageKey ? <Check fontSize="inherit" /> : <ContentCopy fontSize="inherit" />}
                        </IconButton>
                      ) : null}
                    </Stack>
                    {msg.role === 'user' ? (
                      <Typography sx={{ whiteSpace: 'pre-wrap', overflowWrap: 'anywhere', wordBreak: 'break-word' }}>
                        {msg.content}
                      </Typography>
                    ) : (
                      <Box sx={{ minWidth: 0, overflowWrap: 'anywhere', wordBreak: 'break-word' }}>
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
                        {!msg.streaming && msg.role === 'assistant' && msg.usage ? (
                          <HolmesUsageFooter usage={msg.usage} />
                        ) : null}
                      </Box>
                    )}
                  </Box>
                </Box>
              )})}
              </Box>
            )}
            <div ref={messagesEndRef} />
          </Box>
          <Stack sx={{ minWidth: 0, flexShrink: 0 }}>
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
            <Stack
              direction="row"
              justifyContent="space-between"
              alignItems="center"
              flexWrap="wrap"
              gap={1}
              sx={{ mt: 0.75, minWidth: 0, rowGap: 1 }}
            >
              <Stack direction="row" alignItems="center" gap={0.5} sx={{ minWidth: 0, flexWrap: 'wrap' }}>
                <ToggleButtonGroup
                  value={mode}
                  exclusive
                  size="small"
                  onChange={(_, value: 'fast' | 'investigation' | null) => {
                    if (!value || loading) return;
                    setModePersist(value);
                  }}
                  aria-label="Chat mode"
                >
                  <ToggleButton value="fast" disabled={loading}>
                    Fast
                  </ToggleButton>
                  <ToggleButton value="investigation" disabled={loading}>
                    Investigation
                  </ToggleButton>
                </ToggleButtonGroup>
                <Tooltip
                  title={
                    <Box>
                      <Typography variant="subtitle2" sx={{ mb: 0.5 }}>
                        Chat Mode
                      </Typography>
                      <Typography variant="body2">Fast: quick answers, lighter analysis.</Typography>
                      <Typography variant="body2">Investigation: deeper analysis with tools and todo steps.</Typography>
                    </Box>
                  }
                  placement="top"
                >
                  <IconButton size="small" sx={{ color: 'text.secondary' }}>
                    <HelpOutline fontSize="small" />
                  </IconButton>
                </Tooltip>
              </Stack>
              <Box ref={modelAnchorRef} sx={{ flexShrink: 0, minWidth: 0 }}>
                <ButtonGroup variant="contained" size="small" disableElevation>
                  <Button
                    onClick={onSend}
                    disabled={loading || !ask.trim()}
                    startIcon={<Send fontSize="small" />}
                  >
                    {`Send (${selectedModelLabel})`}
                  </Button>
                  <Button
                    size="small"
                    onClick={() => setModelMenuOpen((open) => !open)}
                    disabled={loading}
                    aria-label="Select model"
                  >
                    <ArrowDropDownIcon />
                  </Button>
                </ButtonGroup>
                <Popper open={modelMenuOpen} anchorEl={modelAnchorRef.current} transition placement="top-end">
                  {({ TransitionProps }) => (
                    <Grow {...TransitionProps}>
                      <Paper elevation={6}>
                        <ClickAwayListener onClickAway={() => setModelMenuOpen(false)}>
                          <MenuList autoFocusItem={modelMenuOpen} dense>
                            <MenuItem
                              selected={model === ''}
                              onClick={() => {
                                setModelPersist('');
                                setModelMenuOpen(false);
                              }}
                            >
                              Default
                            </MenuItem>
                            {models.map((m: string) => (
                              <MenuItem
                                key={m}
                                selected={model === m}
                                onClick={() => {
                                  setModelPersist(m);
                                  setModelMenuOpen(false);
                                }}
                              >
                                {m}
                              </MenuItem>
                            ))}
                          </MenuList>
                        </ClickAwayListener>
                      </Paper>
                    </Grow>
                  )}
                </Popper>
              </Box>
            </Stack>
          </Stack>
      </Stack>
    </Box>
  );
};
