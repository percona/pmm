import {
  Alert,
  Box,
  Button,
  CircularProgress,
  ClickAwayListener,
  Collapse,
  Grow,
  IconButton,
  InputBase,
  MenuItem,
  MenuList,
  Paper,
  Popper,
  Stack,
  Tooltip,
  Typography,
  useMediaQuery,
  useTheme,
} from '@mui/material';
import ExpandLess from '@mui/icons-material/ExpandLess';
import ExpandMore from '@mui/icons-material/ExpandMore';
import ContentCopy from '@mui/icons-material/ContentCopy';
import Check from '@mui/icons-material/Check';
import Add from '@mui/icons-material/Add';
import ArrowUpward from '@mui/icons-material/ArrowUpward';
import KeyboardDoubleArrowLeft from '@mui/icons-material/KeyboardDoubleArrowLeft';
import KeyboardDoubleArrowRight from '@mui/icons-material/KeyboardDoubleArrowRight';
import ArrowDropDownIcon from '@mui/icons-material/ArrowDropDown';
import {
  FC,
  ReactNode,
  useState,
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
} from 'react';
import Markdown from 'react-markdown';
import { useSearchParams } from 'react-router-dom';
import remarkGfm from 'remark-gfm';
import rehypeRaw from 'rehype-raw';
import { useAdreModels } from 'hooks/api/useAdre';
import {
  useAdreChat,
  formatTimestamp,
  type ProgressStep,
} from 'hooks/useAdreChat';
import { AdreConversationsSidebar } from './AdreConversationsSidebar';
import { AdreAlertsPanel, type AlertItem } from './AdreAlertsPanel';
import { AdreContextRail } from './AdreContextRail';
import { ResizeHandle } from './ResizeHandle';
import {
  CHATS_WIDTH,
  RAIL_WIDTH,
  useAdrePanelLayout,
} from './useAdrePanelLayout';
import { getMarkdownComponents } from 'components/adre/adre-chat-markdown.helpers';
import { HolmesUsageFooter } from 'components/adre/HolmesUsageFooter';
import {
  loadAdreChatUiPreferences,
  saveAdreChatUiPreferences,
  defaultChatModeFromSettings,
} from 'utils/adreChatUiPreferences';

const PAGE_BG = '#121114';
const CONTENT_WIDTH = 780;

const MODES = [
  {
    value: 'fast',
    label: 'Fast',
    description: 'Quick answers, lighter analysis',
  },
  {
    value: 'investigation',
    label: 'Investigation',
    description: 'Deeper analysis with tools and todo steps',
  },
] as const;

// An answer reads as a document: a highlighted lead, numbered steps, and headings in Poppins.
const answerSx = {
  minWidth: 0,
  fontSize: 15,
  lineHeight: 1.65,
  color: '#E9E7EE',
  overflowWrap: 'anywhere',
  wordBreak: 'break-word',
  '& img': { maxWidth: '100%', height: 'auto', display: 'block' },
  '& h1, & h2, & h3, & h4, & h5, & h6': {
    fontFamily: 'Poppins, sans-serif',
    fontWeight: 600,
    color: '#F4F3F6',
    lineHeight: 1.4,
    mt: 3,
    mb: 1,
  },
  '& h1': { fontSize: 20 },
  '& h2': { fontSize: 18 },
  '& h3, & h4, & h5, & h6': { fontSize: 16 },
  '& > :first-child': { mt: 0 },
  '& strong': { color: '#F4F3F6' },
  '& ul': { pl: 2.75, my: 1 },
  '& li': { my: 0.75 },
  '& ol': { listStyle: 'none', pl: 0, my: 1.5 },
  '& ol > li': { position: 'relative', pl: '40px', my: 2 },
  '& ol > li::before': {
    content: 'attr(data-n)',
    position: 'absolute',
    left: 0,
    top: 0,
    width: 26,
    height: 26,
    borderRadius: '50%',
    bgcolor: '#26252B',
    color: '#CFCBFF',
    fontSize: 13,
    fontWeight: 500,
    lineHeight: '26px',
    textAlign: 'center',
  },
  // The lead: the opening paragraph, or the one under an opening heading such as Summary.
  '& > .adre-md-p:first-child, & > :is(h1, h2, h3):first-child + .adre-md-p': {
    px: 2.25,
    py: 2,
    mb: 2.5,
    borderRadius: '14px',
    bgcolor: 'rgba(182,178,255,0.08)',
    border: '1px solid rgba(182,178,255,0.2)',
    color: '#ECEAF1',
  },
  '& blockquote': {
    m: 0,
    my: 1.5,
    pl: 2,
    color: '#A9A7B0',
    borderLeft: '3px solid rgba(255,255,255,0.12)',
  },
} as const;

// A question reads as a heading; pasted logs and long messages stay body text.
const isShortQuestion = (text: string) =>
  text.length <= 200 && !text.includes('\n');

/** Tool steps of an answer, running or done. */
const StepsList: FC<{ steps: ProgressStep[]; working: boolean }> = ({
  steps,
  working,
}) => (
  <Stack component="ul" gap={0.75} sx={{ m: 0, p: 0, listStyle: 'none' }}>
    {steps.map((step) => (
      <Stack
        component="li"
        key={step.id}
        direction="row"
        gap={1}
        sx={{ fontSize: 13, color: '#A9A7B0' }}
      >
        <Box sx={{ flex: '0 0 16px', mt: '2px', display: 'flex' }}>
          {step.status === 'running' && working ? (
            <CircularProgress size={13} aria-label="Running" />
          ) : (
            <Check aria-label="Done" sx={{ fontSize: 16, color: '#86EFAC' }} />
          )}
        </Box>
        <Box sx={{ minWidth: 0, overflowWrap: 'anywhere' }}>
          <Box
            component="span"
            sx={{ fontFamily: 'Roboto Mono, monospace', color: '#D9D7DE' }}
          >
            {step.toolName}
          </Box>
          {step.description ? ` · ${step.description}` : ''}
        </Box>
      </Stack>
    ))}
  </Stack>
);

/** What is left of a collapsed side panel: a thin strip at the page's edge. */
const EdgeStrip: FC<{ side: 'left' | 'right'; children: ReactNode }> = ({
  side,
  children,
}) => (
  <Stack
    alignItems="center"
    gap={1}
    sx={{
      flex: '0 0 52px',
      py: 2,
      bgcolor: '#17161A',
      [side === 'left' ? 'borderRight' : 'borderLeft']:
        '1px solid rgba(255,255,255,0.06)',
    }}
  >
    {children}
  </Stack>
);

export interface AdreChatPanelProps {
  /** Firing alerts, shown in the side panel or folded above the chat on narrow screens. */
  alerts?: AlertItem[];
}

export const AdreChatPanel: FC<AdreChatPanelProps> = ({ alerts = [] }) => {
  const [searchParams] = useSearchParams();
  // A question from another page starts its own chat instead of joining the last one.
  const [startFresh] = useState(() => searchParams.has('ask'));
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
  } = useAdreChat({ startFresh });
  // Other pages link here with ?ask= to suggest a question; it is not sent until the user does.
  const [ask, setAsk] = useState(() => searchParams.get('ask') ?? '');
  const [model, setModel] = useState('');
  const [mode, setMode] = useState<'fast' | 'investigation'>(() => {
    const p = loadAdreChatUiPreferences();
    if (p.mode === 'fast' || p.mode === 'investigation') return p.mode;
    return 'investigation';
  });
  const [modelMenuOpen, setModelMenuOpen] = useState(false);
  const [expandedReasoningIdx, setExpandedReasoningIdx] = useState<
    number | null
  >(null);
  const [expandedProgressIdx, setExpandedProgressIdx] = useState<number | null>(
    null
  );
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
    setMode(
      defaultChatModeFromSettings(typeof dm === 'string' ? dm : undefined)
    );
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

  useEffect(
    () => () => {
      if (copyResetTimerRef.current != null) {
        window.clearTimeout(copyResetTimerRef.current);
      }
    },
    []
  );

  const copyAssistantMessage = useCallback(
    async (key: string, text: string) => {
      if (!text.trim()) return;
      try {
        await navigator.clipboard.writeText(text);
        setCopiedMessageKey(key);
        if (copyResetTimerRef.current != null) {
          window.clearTimeout(copyResetTimerRef.current);
        }
        copyResetTimerRef.current = window.setTimeout(() => {
          setCopiedMessageKey((prev: string | null) =>
            prev === key ? null : prev
          );
        }, 1600);
      } catch {
        setCopiedMessageKey(null);
      }
    },
    []
  );

  const theme = useTheme();
  // Side by side from md; the side panel needs room, so narrower screens keep each answer's steps with it.
  const wide = useMediaQuery(theme.breakpoints.up('md'));
  const railVisible = useMediaQuery(theme.breakpoints.up('lg'));
  const [layout, setLayout] = useAdrePanelLayout();
  const firingLabel = `${alerts.length} firing ${alerts.length === 1 ? 'alert' : 'alerts'}`;
  const [alertsOpen, setAlertsOpen] = useState(false);
  const latestAssistantIdx = allMessages
    .map((m) => m.role)
    .lastIndexOf('assistant');
  const latest =
    latestAssistantIdx >= 0 ? allMessages[latestAssistantIdx] : undefined;
  const railSteps = latest?.streaming
    ? progressSteps
    : (latest?.progressSteps ?? []);

  return (
    <Box
      sx={{
        flex: 1,
        minHeight: 0,
        width: '100%',
        minWidth: 0,
        display: 'flex',
        flexDirection: { xs: 'column', md: 'row' },
        overflow: 'hidden',
        bgcolor: PAGE_BG,
        color: '#F4F3F6',
      }}
    >
      {wide && !layout.chatsOpen ? (
        <EdgeStrip side="left">
          <Tooltip title="Expand chats" placement="right">
            <IconButton
              aria-label="Expand chats"
              onClick={() => setLayout({ chatsOpen: true })}
              sx={{ color: '#A9A7B0' }}
            >
              <KeyboardDoubleArrowRight fontSize="small" />
            </IconButton>
          </Tooltip>
          <Tooltip title="New chat" placement="right">
            <IconButton
              aria-label="New chat"
              onClick={() => void newChat()}
              sx={{
                bgcolor: 'rgba(182,178,255,0.14)',
                color: '#CFCBFF',
                '&:hover': { bgcolor: 'rgba(182,178,255,0.22)' },
              }}
            >
              <Add fontSize="small" />
            </IconButton>
          </Tooltip>
        </EdgeStrip>
      ) : (
        <>
          <Box
            sx={{
              flex: { xs: '0 0 auto', md: `0 0 ${layout.chatsWidth}px` },
              width: { md: layout.chatsWidth },
              maxHeight: { xs: 'min(36vh, 260px)', md: 'none' },
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
              onCollapse={
                wide ? () => setLayout({ chatsOpen: false }) : undefined
              }
            />
          </Box>
          {wide && (
            <ResizeHandle
              side="left"
              width={layout.chatsWidth}
              min={CHATS_WIDTH.min}
              max={CHATS_WIDTH.max}
              onResize={(chatsWidth) => setLayout({ chatsWidth })}
              label="Resize chats"
            />
          )}
        </>
      )}

      <Stack sx={{ flex: 1, minHeight: 0, minWidth: 0, overflow: 'hidden' }}>
        {!railVisible && alerts.length > 0 && (
          <Box
            sx={{
              px: { xs: 2, md: 5 },
              pt: 2,
              pb: alertsOpen ? 2 : 0,
              flexShrink: 0,
              borderBottom: alertsOpen ? 1 : 0,
              borderColor: 'rgba(255,255,255,0.08)',
            }}
          >
            <Button
              size="small"
              aria-expanded={alertsOpen}
              onClick={() => setAlertsOpen((open) => !open)}
              endIcon={alertsOpen ? <ExpandLess /> : <ExpandMore />}
              sx={{
                borderRadius: 999,
                px: 1.75,
                bgcolor: 'rgba(248,113,113,0.14)',
                color: '#FCA5A5',
                '&:hover': { bgcolor: 'rgba(248,113,113,0.2)' },
              }}
            >
              {firingLabel}
            </Button>
            <Collapse in={alertsOpen}>
              <Box
                sx={{
                  pt: 1.5,
                  maxWidth: 520,
                  maxHeight: '45vh',
                  overflowY: 'auto',
                }}
              >
                <AdreAlertsPanel alerts={alerts} />
              </Box>
            </Collapse>
          </Box>
        )}
        {chatError ? (
          <Alert severity="error" sx={{ mx: { xs: 2, md: 5 }, mt: 2 }}>
            {chatError}
          </Alert>
        ) : null}
        <Box
          ref={containerRef}
          id="messages-container"
          sx={{
            flex: 1,
            minHeight: 0,
            overflowY: 'auto',
            overflowX: 'hidden',
            px: { xs: 2, md: 5 },
            py: 4,
          }}
        >
          <Stack
            gap={3}
            sx={{ maxWidth: CONTENT_WIDTH, mx: 'auto', minWidth: 0 }}
          >
            {allMessages.length === 0 ? (
              <Typography
                sx={{
                  alignSelf: 'center',
                  mt: 6,
                  color: '#A9A7B0',
                  fontSize: 15,
                }}
              >
                Ask a question about your database environment...
              </Typography>
            ) : (
              allMessages.map((msg, idx) => {
                const messageKey = String(msg.serverMessageId ?? `row-${idx}`);
                const messageText = (msg.content || response || '').trim();
                const inRail =
                  railVisible && layout.railOpen && idx === latestAssistantIdx;
                const time = msg.timestamp
                  ? ` · ${formatTimestamp(msg.timestamp)}`
                  : '';
                if (msg.role === 'user') {
                  return (
                    <Box
                      key={messageKey}
                      data-adre-msg-id={
                        msg.serverMessageId != null
                          ? String(msg.serverMessageId)
                          : undefined
                      }
                      sx={{
                        pt: idx > 0 ? 3 : 0,
                        borderTop: idx > 0 ? 1 : 0,
                        borderColor: 'rgba(255,255,255,0.08)',
                      }}
                    >
                      <Typography sx={{ fontSize: 12, color: '#8E8C95' }}>
                        {`You${time}`}
                      </Typography>
                      <Typography
                        sx={
                          isShortQuestion(msg.content)
                            ? {
                                mt: 0.75,
                                fontFamily: 'Poppins, sans-serif',
                                fontSize: 18,
                                fontWeight: 500,
                                lineHeight: 1.45,
                                overflowWrap: 'anywhere',
                              }
                            : {
                                mt: 0.75,
                                fontSize: 15,
                                lineHeight: 1.6,
                                whiteSpace: 'pre-wrap',
                                overflowWrap: 'anywhere',
                              }
                        }
                      >
                        {msg.content}
                      </Typography>
                    </Box>
                  );
                }
                return (
                  <Box
                    key={messageKey}
                    component="article"
                    data-adre-msg-id={
                      msg.serverMessageId != null
                        ? String(msg.serverMessageId)
                        : undefined
                    }
                    sx={{ minWidth: 0 }}
                  >
                    <Stack
                      direction="row"
                      alignItems="center"
                      justifyContent="space-between"
                      sx={{ mb: 1 }}
                    >
                      <Typography sx={{ fontSize: 12, color: '#8E8C95' }}>
                        {`Assistant${time}`}
                      </Typography>
                      {messageText ? (
                        <IconButton
                          size="small"
                          aria-label={
                            copiedMessageKey === messageKey
                              ? 'Copied'
                              : 'Copy response'
                          }
                          title={
                            copiedMessageKey === messageKey
                              ? 'Copied'
                              : 'Copy response'
                          }
                          onClick={() =>
                            copyAssistantMessage(messageKey, messageText)
                          }
                          sx={{
                            color:
                              copiedMessageKey === messageKey
                                ? 'success.light'
                                : '#A9A7B0',
                          }}
                        >
                          {copiedMessageKey === messageKey ? (
                            <Check fontSize="inherit" />
                          ) : (
                            <ContentCopy fontSize="inherit" />
                          )}
                        </IconButton>
                      ) : null}
                    </Stack>
                    {(msg.reasoning ?? (msg.streaming && reasoning)) && (
                      <Box sx={{ mb: 1.5 }}>
                        <Button
                          size="small"
                          onClick={() =>
                            setExpandedReasoningIdx((prev: number | null) =>
                              prev === idx ? null : idx
                            )
                          }
                          startIcon={
                            expandedReasoningIdx === idx ? (
                              <ExpandLess />
                            ) : (
                              <ExpandMore />
                            )
                          }
                          sx={{ color: '#A9A7B0', px: 1 }}
                        >
                          Reasoning
                        </Button>
                        <Collapse in={expandedReasoningIdx === idx}>
                          <Typography
                            sx={{
                              mt: 1,
                              fontSize: 14,
                              color: '#A9A7B0',
                              fontStyle: 'italic',
                              whiteSpace: 'pre-wrap',
                            }}
                          >
                            {msg.reasoning ?? reasoning}
                          </Typography>
                        </Collapse>
                      </Box>
                    )}
                    {!inRail && msg.streaming && progressSteps.length > 0 && (
                      <Box sx={{ mb: 1.5 }}>
                        <StepsList steps={progressSteps} working={loading} />
                      </Box>
                    )}
                    {!inRail &&
                      !msg.streaming &&
                      (msg.progressSteps?.length ?? 0) > 0 && (
                        <Box sx={{ mb: 1.5 }}>
                          <Button
                            size="small"
                            onClick={() =>
                              setExpandedProgressIdx((prev: number | null) =>
                                prev === idx ? null : idx
                              )
                            }
                            startIcon={
                              expandedProgressIdx === idx ? (
                                <ExpandLess />
                              ) : (
                                <ExpandMore />
                              )
                            }
                            sx={{ color: '#A9A7B0', px: 1 }}
                          >
                            {`Steps · ${msg.progressSteps?.length}`}
                          </Button>
                          <Collapse in={expandedProgressIdx === idx}>
                            <Box sx={{ pt: 1 }}>
                              <StepsList
                                steps={msg.progressSteps ?? []}
                                working={false}
                              />
                            </Box>
                          </Collapse>
                        </Box>
                      )}
                    {messageText ? (
                      <Box sx={answerSx}>
                        <Markdown
                          remarkPlugins={[remarkGfm]}
                          rehypePlugins={[rehypeRaw]}
                          components={getMarkdownComponents(
                            msg.content || response || ''
                          )}
                        >
                          {msg.content || response}
                        </Markdown>
                      </Box>
                    ) : msg.streaming && loading && !response ? (
                      <Typography sx={{ fontSize: 14, color: '#A9A7B0' }}>
                        {progressSteps.length > 0 ? 'Working…' : 'Typing...'}
                      </Typography>
                    ) : null}
                    {!msg.streaming && msg.usage ? (
                      <HolmesUsageFooter usage={msg.usage} align="left" />
                    ) : null}
                  </Box>
                );
              })
            )}
            <div ref={messagesEndRef} />
          </Stack>
        </Box>

        <Box
          component="form"
          onSubmit={(e: React.FormEvent) => {
            e.preventDefault();
            void onSend();
          }}
          sx={{ flexShrink: 0, px: { xs: 2, md: 5 }, pb: 3, pt: 1 }}
        >
          <Box
            sx={{
              maxWidth: CONTENT_WIDTH,
              mx: 'auto',
              p: '12px 10px 8px 16px',
              borderRadius: '16px',
              bgcolor: '#1D1C21',
              border: 1,
              borderColor: 'rgba(255,255,255,0.1)',
            }}
          >
            <InputBase
              multiline
              fullWidth
              minRows={2}
              maxRows={8}
              placeholder="Message the Assistant..."
              value={ask}
              onChange={(e) => setAsk(e.target.value)}
              onKeyDown={(e: React.KeyboardEvent) => {
                if (e.key === 'Enter' && !e.shiftKey) {
                  e.preventDefault();
                  void onSend();
                }
              }}
              inputProps={{ 'aria-label': 'Message the Assistant' }}
              sx={{ fontSize: 15, lineHeight: 1.5 }}
            />
            <Stack direction="row" alignItems="center" gap={1} sx={{ mt: 1 }}>
              <Box
                role="group"
                aria-label="Chat mode"
                sx={{
                  display: 'flex',
                  p: '3px',
                  borderRadius: 999,
                  bgcolor: '#141317',
                }}
              >
                {MODES.map((m) => (
                  <Tooltip key={m.value} title={m.description}>
                    <span>
                      <Button
                        size="small"
                        aria-pressed={mode === m.value}
                        disabled={loading}
                        onClick={() => setModePersist(m.value)}
                        sx={{
                          minWidth: 0,
                          height: 28,
                          px: 1.5,
                          fontSize: 13,
                          fontWeight: mode === m.value ? 500 : 400,
                          color: mode === m.value ? '#F4F3F6' : '#A9A7B0',
                          bgcolor: mode === m.value ? '#2E2C36' : 'transparent',
                          '&:hover': {
                            bgcolor:
                              mode === m.value
                                ? '#2E2C36'
                                : 'rgba(255,255,255,0.05)',
                          },
                        }}
                      >
                        {m.label}
                      </Button>
                    </span>
                  </Tooltip>
                ))}
              </Box>
              <Box ref={modelAnchorRef}>
                <Button
                  size="small"
                  onClick={() => setModelMenuOpen((open) => !open)}
                  disabled={loading}
                  aria-label="Select model"
                  endIcon={<ArrowDropDownIcon />}
                  sx={{
                    height: 32,
                    fontSize: 13,
                    fontWeight: 400,
                    color: '#A9A7B0',
                  }}
                >
                  {model || 'Default model'}
                </Button>
              </Box>
              <Box sx={{ flex: 1 }} />
              <IconButton
                type="submit"
                aria-label="Send"
                disabled={loading || !ask.trim()}
                sx={{
                  width: 38,
                  height: 38,
                  bgcolor: 'primary.main',
                  color: 'primary.contrastText',
                  '&:hover': { bgcolor: 'primary.dark' },
                  '&.Mui-disabled': {
                    bgcolor: 'rgba(255,255,255,0.08)',
                    color: 'rgba(255,255,255,0.3)',
                  },
                }}
              >
                <ArrowUpward fontSize="small" />
              </IconButton>
            </Stack>
          </Box>
          <Popper
            open={modelMenuOpen}
            anchorEl={modelAnchorRef.current}
            transition
            placement="top-end"
          >
            {({ TransitionProps }) => (
              <Grow {...TransitionProps}>
                <Paper elevation={6}>
                  <ClickAwayListener
                    onClickAway={() => setModelMenuOpen(false)}
                  >
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

      {railVisible &&
        (layout.railOpen ? (
          <>
            <ResizeHandle
              side="right"
              width={layout.railWidth}
              min={RAIL_WIDTH.min}
              max={RAIL_WIDTH.max}
              onResize={(railWidth) => setLayout({ railWidth })}
              label="Resize side panel"
            />
            <Box
              component="aside"
              aria-label="Answer details"
              sx={{
                flex: `0 0 ${layout.railWidth}px`,
                width: layout.railWidth,
                minHeight: 0,
                overflowY: 'auto',
                boxSizing: 'border-box',
                px: 2.5,
                pt: 1.5,
                pb: 3,
                bgcolor: '#17161A',
              }}
            >
              <Tooltip title="Collapse side panel">
                <IconButton
                  aria-label="Collapse side panel"
                  onClick={() => setLayout({ railOpen: false })}
                  sx={{ ml: -1, mb: 0.5, color: '#A9A7B0' }}
                >
                  <KeyboardDoubleArrowRight fontSize="small" />
                </IconButton>
              </Tooltip>
              <AdreContextRail
                alerts={alerts}
                steps={railSteps}
                working={!!latest?.streaming && loading}
              />
            </Box>
          </>
        ) : (
          <EdgeStrip side="right">
            <Tooltip title="Expand side panel" placement="left">
              <IconButton
                aria-label={
                  alerts.length > 0
                    ? `Expand side panel (${firingLabel})`
                    : 'Expand side panel'
                }
                onClick={() => setLayout({ railOpen: true })}
                sx={{ color: '#A9A7B0' }}
              >
                <KeyboardDoubleArrowLeft fontSize="small" />
              </IconButton>
            </Tooltip>
            {alerts.length > 0 && (
              <Box
                aria-hidden
                sx={{
                  minWidth: 22,
                  px: 0.75,
                  borderRadius: 999,
                  bgcolor: 'rgba(248,113,113,0.14)',
                  color: '#FCA5A5',
                  fontSize: 12,
                  fontWeight: 500,
                  textAlign: 'center',
                }}
              >
                {alerts.length}
              </Box>
            )}
          </EdgeStrip>
        ))}
    </Box>
  );
};
