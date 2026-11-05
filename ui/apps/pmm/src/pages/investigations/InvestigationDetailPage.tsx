import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  CircularProgress,
  IconButton,
  Link,
  MenuItem,
  Select,
  Stack,
  Snackbar,
  Tooltip,
  Typography,
} from '@mui/material';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import DataObjectIcon from '@mui/icons-material/DataObject';
import PictureAsPdfIcon from '@mui/icons-material/PictureAsPdf';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import ArrowUpwardIcon from '@mui/icons-material/ArrowUpward';
import ArrowDownwardIcon from '@mui/icons-material/ArrowDownward';
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline';
import { FC, Fragment, useState, useEffect, useRef } from 'react';
import { Link as RouterLink, useParams, useNavigate } from 'react-router-dom';
import { useQueryClient } from '@tanstack/react-query';
import { Page } from 'components/page';
import {
  useInvestigation,
  useInvestigationComments,
  useInvestigationMessages,
  useInvestigationTimeline,
  useSimilarInvestigations,
  usePostInvestigationComment,
  usePostInvestigationChat,
  usePostInvestigationRun,
  usePatchInvestigation,
  usePatchInvestigationBlock,
  useDeleteInvestigationBlock,
  useCreateServiceNowTicket,
  INVESTIGATIONS_KEYS,
} from 'hooks/api/useInvestigations';
import { useAdreSettings } from 'hooks/api/useAdre';
import { useInvestigationUsage } from 'hooks/api/useAdreUsage';
import { PMM_NEW_NAV_PATH } from 'lib/constants';
import { getInvestigationExportPdfUrl } from 'api/investigations';
import type { Investigation, InvestigationBlock } from 'api/investigations';
import {
  getAdreAlerts,
  getAlertMetadataFromLabels,
  type AlertMetadataFromLabels,
} from 'api/adre';
import {
  aggregateInvestigationUsage,
  formatTokensWithCached,
  formatUsdCost,
  HOLMES_FEATURE_LABELS,
} from 'utils/holmesUsageFormat';
import { BlockRenderer } from './components/BlockRenderer';
import { TimelineSection } from './components/TimelineSection';
import { SeenBefore } from './components/SeenBefore';
import { InvestigationLead } from './components/InvestigationLead';
import {
  InvestigationChat,
  InvestigationComments,
} from './components/InvestigationConversation';
import { StatusPill } from './components/StatusPill';
import { adreColors, headingFont } from 'components/adre/adreAnswerSx';
import {
  accentButtonSx,
  pillButtonSx,
  sourceLabel,
} from './investigationStyles';

const STATUS_OPTIONS = [
  'open',
  'in_progress',
  'investigating',
  'running',
  'completed',
  'failed',
  'resolved',
  'archived',
] as const;

const railCardSx = {
  p: 2.25,
  borderRadius: '16px',
  bgcolor: adreColors.panel,
  border: `1px solid ${adreColors.divider}`,
} as const;

const railTitleSx = {
  fontFamily: headingFont,
  fontSize: 14,
  fontWeight: 600,
  mb: 1.25,
} as const;

const railActionSx = {
  justifyContent: 'flex-start',
  px: 1,
  fontSize: 13,
  fontWeight: 400,
  textTransform: 'none',
  color: adreColors.body,
  '& .MuiButton-startIcon': { color: adreColors.secondary },
} as const;

function investigationUserRequest(inv: Investigation): string {
  const fromApi = (inv.userRequest ?? inv.user_request ?? '').trim();
  if (fromApi) {
    return fromApi;
  }
  const summary = (inv.summary ?? '').trim();
  if (!summary) {
    return '';
  }
  const hasReport =
    !!inv.rootCauseSummary?.trim() ||
    !!inv.summaryDetailed?.trim() ||
    (inv.blocks?.length ?? 0) > 0 ||
    ['completed', 'resolved', 'failed'].includes(inv.status);
  return hasReport ? '' : summary;
}

const BlockWithActions: FC<{
  block: InvestigationBlock;
  index: number;
  total: number;
  onMoveUp: () => void;
  onMoveDown: () => void;
  onDelete: () => void;
  isPending: boolean;
}> = ({ block, index, total, onMoveUp, onMoveDown, onDelete, isPending }) => (
  <Box
    sx={{
      position: 'relative',
      py: 2.5,
      borderTop: index > 0 ? `1px solid ${adreColors.divider}` : 0,
      '&:hover .block-actions, &:focus-within .block-actions': { opacity: 1 },
    }}
  >
    <BlockRenderer block={block} />
    <Stack
      className="block-actions"
      direction="row"
      sx={{
        position: 'absolute',
        top: 16,
        right: 0,
        opacity: 0,
        transition: 'opacity 120ms',
        '@media (hover: none)': { opacity: 1 },
        '& .MuiIconButton-root': { color: adreColors.secondary },
      }}
    >
      <IconButton
        size="small"
        aria-label="Move block up"
        onClick={onMoveUp}
        disabled={index === 0 || isPending}
      >
        <ArrowUpwardIcon fontSize="small" />
      </IconButton>
      <IconButton
        size="small"
        aria-label="Move block down"
        onClick={onMoveDown}
        disabled={index >= total - 1 || isPending}
      >
        <ArrowDownwardIcon fontSize="small" />
      </IconButton>
      <IconButton
        size="small"
        aria-label="Delete block"
        onClick={onDelete}
        disabled={isPending}
      >
        <DeleteOutlineIcon fontSize="small" />
      </IconButton>
    </Stack>
  </Box>
);

const InvestigationDetailPage: FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [isRunning, setIsRunning] = useState(false);
  const {
    data: inv,
    isLoading,
    isError,
    error,
  } = useInvestigation(id, {
    refetchInterval: isRunning ? 5000 : false,
  });
  const { data: comments = [] } = useInvestigationComments(id);
  const { data: messages = [] } = useInvestigationMessages(
    id,
    { limit: 50 },
    { refetchInterval: isRunning ? 5000 : false }
  );
  const { data: timelineEvents = [] } = useInvestigationTimeline(id);
  const { data: similar = [] } = useSimilarInvestigations(id);
  const postComment = usePostInvestigationComment(id ?? '');
  const postChat = usePostInvestigationChat(id ?? '');
  const postRun = usePostInvestigationRun(id ?? '');
  const patchInv = usePatchInvestigation(id ?? '');
  const patchBlock = usePatchInvestigationBlock(id ?? '');
  const deleteBlock = useDeleteInvestigationBlock(id ?? '');
  const createSNTicket = useCreateServiceNowTicket(id ?? '');
  const { data: adreSettings } = useAdreSettings();
  const { data: usageBreakdown, isLoading: usageLoading } =
    useInvestigationUsage(id, {
      refetchInterval: isRunning ? 5000 : false,
    });
  const [commentText, setCommentText] = useState('');
  const [chatText, setChatText] = useState('');
  const [copyDone, setCopyDone] = useState(false);
  const [snackMessage, setSnackMessage] = useState<string | null>(null);
  const [snackSeverity, setSnackSeverity] = useState<'error' | 'success'>(
    'error'
  );
  const [fetchedAlertMeta, setFetchedAlertMeta] =
    useState<AlertMetadataFromLabels>({});
  const prevStatusRef = useRef<string | undefined>(undefined);

  const showError = (msg: string) => {
    setSnackMessage(msg);
    setSnackSeverity('error');
  };
  const showSuccess = (msg: string) => {
    setSnackMessage(msg);
    setSnackSeverity('success');
  };

  useEffect(() => {
    if (!inv || inv.id !== id) return;
    const status = inv.status;
    const prev = prevStatusRef.current;
    prevStatusRef.current = status;
    setIsRunning(status === 'running');
    if (prev === 'running' && status === 'completed') {
      showSuccess('Investigation completed');
    } else if (prev === 'running' && status === 'failed') {
      showError('Investigation failed');
    }
    if (
      prev === 'running' &&
      (status === 'completed' || status === 'failed') &&
      id
    ) {
      void queryClient.invalidateQueries({
        queryKey: INVESTIGATIONS_KEYS.detail(id),
      });
      void queryClient.invalidateQueries({
        queryKey: INVESTIGATIONS_KEYS.messagesPrefix(id),
      });
      void queryClient.invalidateQueries({
        queryKey: INVESTIGATIONS_KEYS.usage(id),
      });
    }
  }, [inv, id, queryClient]);

  const invId = inv?.id;
  const invSourceType = inv?.sourceType;
  const invSourceRef = inv?.sourceRef;
  // When investigation is from an alert but API didn't return node/service, fetch alerts and derive metadata
  useEffect(() => {
    if (invId == null) return;
    setFetchedAlertMeta({});
    if (invSourceType !== 'alert' || !invSourceRef) return;
    const refs = new Set(
      invSourceRef
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean)
    );
    if (refs.size === 0) return;
    let cancelled = false;
    getAdreAlerts()
      .then((data: unknown) => {
        if (cancelled) return;
        const raw = data as {
          data?: {
            alerts?: Array<{
              fingerprint?: string;
              labels?: Record<string, string>;
            }>;
          };
          alerts?: Array<{
            fingerprint?: string;
            labels?: Record<string, string>;
          }>;
        };
        const list = raw?.data?.alerts ?? raw?.alerts ?? [];
        const arr = Array.isArray(list) ? list : [];
        const match = arr.find((a) => a.fingerprint && refs.has(a.fingerprint));
        if (match?.labels) {
          setFetchedAlertMeta(getAlertMetadataFromLabels(match.labels));
        }
      })
      .catch(() => {});
    return () => {
      cancelled = true;
    };
  }, [invId, invSourceType, invSourceRef]);

  const getErrorMessage = (err: unknown): string => {
    const ax = err as { response?: { data?: { error?: string } } };
    return (
      ax?.response?.data?.error ?? (err as Error)?.message ?? 'Request failed'
    );
  };

  const handleCopyLink = () => {
    const url = `${window.location.origin}${window.location.pathname}`;
    void navigator.clipboard.writeText(url).then(() => {
      setCopyDone(true);
      setTimeout(() => setCopyDone(false), 2000);
    });
  };

  const handleAddComment = () => {
    if (!commentText.trim() || !id) return;
    postComment.mutate(
      { content: commentText.trim() },
      {
        onSuccess: () => setCommentText(''),
      }
    );
  };

  const handleCopyMarkdown = async () => {
    if (!inv) return;
    const blocks = [...(inv.blocks ?? [])].sort(
      (a, b) => a.position - b.position
    );
    const blockText = blocks
      .map((b) => {
        const content = b.dataJson as
          | { content?: string; steps?: string[] }
          | undefined;
        if (content?.content)
          return `## ${b.title || b.type}\n${content.content}`;
        if (Array.isArray(content?.steps)) {
          return `## ${b.title || b.type}\n${content.steps.map((s, i) => `${i + 1}. ${s}`).join('\n')}`;
        }
        return '';
      })
      .filter(Boolean)
      .join('\n\n');
    const md = [
      `# ${inv.title || 'Investigation'}`,
      `Status: ${inv.status}`,
      `Confidence: ${inv.confidence} (${inv.confidenceScore ?? 0})`,
      inv.summary ? `\n## Summary\n${inv.summary}` : '',
      inv.rootCauseSummary ? `\n## Root cause\n${inv.rootCauseSummary}` : '',
      inv.resolutionSummary ? `\n## Resolution\n${inv.resolutionSummary}` : '',
      blockText ? `\n## Report\n${blockText}` : '',
    ]
      .filter(Boolean)
      .join('\n');
    await navigator.clipboard.writeText(md);
    showSuccess('Copied markdown');
  };

  const handleCopyEvidenceJson = async () => {
    if (!inv) return;
    const payload = {
      id: inv.id,
      title: inv.title,
      status: inv.status,
      confidence: inv.confidence,
      confidenceScore: inv.confidenceScore ?? 0,
      confidenceRationale: inv.confidenceRationale ?? '',
      evidence: inv.evidence ?? [],
      summary: inv.summary,
      rootCauseSummary: inv.rootCauseSummary,
      resolutionSummary: inv.resolutionSummary,
      timeFrom: inv.timeFrom,
      timeTo: inv.timeTo,
    };
    await navigator.clipboard.writeText(JSON.stringify(payload, null, 2));
    showSuccess('Copied JSON evidence bundle');
  };

  const handleSendChat = () => {
    if (!chatText.trim() || !id) return;
    postChat.mutate(chatText.trim(), {
      onSuccess: () => setChatText(''),
      onError: (err) => showError(`Chat failed: ${getErrorMessage(err)}`),
    });
  };

  if (isLoading || !id) {
    return (
      <Page title="Investigation">
        <Box display="flex" justifyContent="center" p={4}>
          <CircularProgress />
        </Box>
      </Page>
    );
  }

  if (isError || !inv) {
    return (
      <Page title="Investigation">
        <Card variant="outlined">
          <CardContent>
            <Alert severity="error">
              {inv
                ? 'Failed to load investigation.'
                : 'Investigation not found.'}
              {(error as Error)?.message && ` ${(error as Error).message}`}
            </Alert>
            <Button
              startIcon={<ArrowBackIcon />}
              onClick={() => navigate(`${PMM_NEW_NAV_PATH}/investigations`)}
              sx={{ mt: 2 }}
            >
              Back to list
            </Button>
          </CardContent>
        </Card>
      </Page>
    );
  }

  // The block-based report already contains a "Summary" section plus root cause / resolution, so the
  // scalar summary cards are only shown as a fallback when there are no report blocks (avoids the
  // duplicate "Summary" at the top and "Detailed summary" at the bottom).
  const hasReportBlocks = (inv.blocks?.length ?? 0) > 0;

  const timeFrom = inv.timeFrom ?? (inv as { time_from?: string }).time_from;
  const timeTo = inv.timeTo ?? (inv as { time_to?: string }).time_to;
  const timeRange =
    timeFrom && timeTo
      ? Date.parse(timeFrom) === Date.parse(timeTo)
        ? new Date(timeFrom).toLocaleString()
        : `${new Date(timeFrom).toLocaleString()} — ${new Date(timeTo).toLocaleString()}`
      : null;

  const usageSummary = aggregateInvestigationUsage({
    holmesCallCount: inv.holmesCallCount,
    holmes_call_count: inv.holmes_call_count,
    holmesTotalTokens: inv.holmesTotalTokens,
    holmes_total_tokens: inv.holmes_total_tokens,
    holmesTotalCost: inv.holmesTotalCost,
    holmes_total_cost: inv.holmes_total_cost,
    messages,
    events: usageBreakdown?.events,
  });

  const showUsageLoading = usageLoading && !usageSummary.hasUsage;

  const userRequest = investigationUserRequest(inv);
  // A new investigation keeps its request as its summary; show it once.
  const leadSummary =
    userRequest && (inv.summary ?? '').trim() === userRequest
      ? ''
      : (inv.summary ?? '');
  const sortedBlocks = [...(inv.blocks ?? [])].sort(
    (a, b) => a.position - b.position
  );
  const nodeName =
    inv.nodeName ??
    (inv as { node_name?: string }).node_name ??
    fetchedAlertMeta.nodeName;
  const serviceName =
    inv.serviceName ??
    (inv as { service_name?: string }).service_name ??
    fetchedAlertMeta.serviceName;
  const clusterName =
    inv.clusterName ??
    (inv as { cluster_name?: string }).cluster_name ??
    fetchedAlertMeta.clusterName;
  const severity = inv.severity || fetchedAlertMeta.severity;
  const source = sourceLabel(inv.sourceType);
  // The server refuses to re-run a finished investigation.
  const runBlocked = ['completed', 'failed', 'resolved', 'archived'].includes(
    inv.status
  );

  const moveBlock = (from: InvestigationBlock, to: InvestigationBlock) =>
    patchBlock.mutate(
      { blockId: from.id, body: { position: to.position } },
      {
        onSuccess: () =>
          patchBlock.mutate({
            blockId: to.id,
            body: { position: from.position },
          }),
      }
    );

  const ticketId = inv.servicenowTicketId ?? inv.servicenow_ticket_id;
  const ticketNumber =
    inv.servicenowTicketNumber ?? inv.servicenow_ticket_number;
  const snConfigured =
    adreSettings?.servicenowConfigured ??
    adreSettings?.servicenow_configured ??
    false;
  const ticketHref = (() => {
    if (!ticketId) return '';
    try {
      const origin = new URL(
        adreSettings?.servicenowUrl ?? adreSettings?.servicenow_url ?? ''
      ).origin;
      return `${origin}/nav_to.do?uri=incident.do?sys_id=${ticketId}`;
    } catch {
      return '';
    }
  })();

  const details = [
    ['Source', source],
    ['Window', timeRange],
    ['Severity', severity],
    ['Node', nodeName],
    ['Service', serviceName],
    ['Cluster', clusterName],
    ['Created', new Date(inv.createdAt).toLocaleString()],
    ['Created by', inv.createdBy],
  ].filter((d): d is [string, string] => !!d[1]);

  return (
    <Page
      title={inv.title || 'Investigation'}
      hideTitle
      fullWidth
      footer={null}
    >
      <Box
        sx={{
          flex: 1,
          minHeight: 0,
          overflowY: 'auto',
          bgcolor: adreColors.page,
          color: adreColors.text,
          borderRadius: 1,
        }}
      >
        <Box
          sx={{
            maxWidth: 1320,
            mx: 'auto',
            px: { xs: 2, md: 4 },
            py: { xs: 2.5, md: 3.5 },
            display: 'flex',
            flexWrap: 'wrap',
            alignItems: 'flex-start',
            gap: 4,
          }}
        >
          <Stack
            component="article"
            gap={4}
            sx={{ flex: '999 1 620px', minWidth: 0, maxWidth: 920 }}
          >
            <Stack component="header" gap={1.5}>
              <Link
                component={RouterLink}
                to={`${PMM_NEW_NAV_PATH}/investigations`}
                sx={{
                  alignSelf: 'flex-start',
                  display: 'inline-flex',
                  alignItems: 'center',
                  gap: 0.5,
                  fontSize: 13,
                  color: adreColors.secondary,
                  textDecoration: 'none',
                  '&:hover': { color: adreColors.text },
                }}
              >
                <ArrowBackIcon sx={{ fontSize: 16 }} />
                Investigations
              </Link>
              <Stack
                direction="row"
                flexWrap="wrap"
                alignItems="flex-start"
                justifyContent="space-between"
                gap={2}
              >
                <Typography
                  component="h1"
                  sx={{
                    flex: '1 1 360px',
                    minWidth: 0,
                    fontFamily: headingFont,
                    fontSize: { xs: 22, md: 26 },
                    fontWeight: 600,
                    lineHeight: 1.3,
                    overflowWrap: 'anywhere',
                  }}
                >
                  {inv.title || 'Investigation'}
                </Typography>
                <Stack direction="row" gap={1} flexWrap="wrap">
                  {ticketId ? (
                    <Chip
                      icon={<CheckCircleOutlineIcon />}
                      label={`ServiceNow: ${ticketNumber || ticketId}`}
                      color="success"
                      size="small"
                      variant="outlined"
                      clickable={!!ticketHref}
                      onClick={
                        ticketHref
                          ? () =>
                              window.open(
                                ticketHref,
                                '_blank',
                                'noopener,noreferrer'
                              )
                          : undefined
                      }
                      sx={{ height: 36, borderRadius: 999 }}
                    />
                  ) : (
                    <Tooltip
                      title={
                        snConfigured
                          ? ''
                          : 'Configure ServiceNow in the Assistant configuration'
                      }
                    >
                      <span>
                        <Button
                          variant="outlined"
                          disabled={!snConfigured || createSNTicket.isPending}
                          onClick={() =>
                            createSNTicket.mutate(undefined, {
                              onError: (err) =>
                                showError(
                                  `ServiceNow: ${getErrorMessage(err)}`
                                ),
                              onSuccess: (data) =>
                                showSuccess(
                                  `ServiceNow ticket created: ${data.ticket_number || data.ticket_id}`
                                ),
                            })
                          }
                          sx={pillButtonSx}
                        >
                          {createSNTicket.isPending
                            ? 'Creating…'
                            : 'Create ServiceNow ticket'}
                        </Button>
                      </span>
                    </Tooltip>
                  )}
                  <Tooltip
                    title={
                      runBlocked
                        ? `This investigation is ${inv.status}; create a new one to analyze again.`
                        : ''
                    }
                  >
                    <span>
                      <Button
                        variant="contained"
                        disableElevation
                        onClick={() =>
                          postRun.mutate(undefined, {
                            onError: (err) =>
                              showError(`Run failed: ${getErrorMessage(err)}`),
                            onSuccess: () => setIsRunning(true),
                          })
                        }
                        disabled={postRun.isPending || isRunning || runBlocked}
                        sx={accentButtonSx}
                      >
                        {isRunning ? 'Running…' : 'Run investigation'}
                      </Button>
                    </span>
                  </Tooltip>
                </Stack>
              </Stack>
              <Stack
                direction="row"
                flexWrap="wrap"
                alignItems="center"
                gap={1.25}
                sx={{ fontSize: 13, color: adreColors.secondary }}
              >
                <StatusPill status={inv.status} running={isRunning} />
                <span>{source}</span>
                {timeRange && <span>· {timeRange}</span>}
                {serviceName && <span>· {serviceName}</span>}
              </Stack>
              {userRequest && (
                <Box sx={{ mt: 1 }}>
                  <Typography sx={{ fontSize: 12, color: adreColors.muted }}>
                    Request
                  </Typography>
                  <Typography
                    sx={{
                      fontSize: 16,
                      color: adreColors.text,
                      whiteSpace: 'pre-wrap',
                      overflowWrap: 'anywhere',
                    }}
                  >
                    {userRequest}
                  </Typography>
                </Box>
              )}
            </Stack>

            <InvestigationLead
              inv={inv}
              summary={leadSummary}
              detail={hasReportBlocks ? undefined : inv.summaryDetailed}
              running={isRunning}
            />

            <SeenBefore items={similar} />

            <TimelineSection events={timelineEvents} />

            {sortedBlocks.length > 0 && (
              <Box component="section" aria-label="Report">
                <Typography
                  component="h2"
                  sx={{
                    fontFamily: headingFont,
                    fontSize: 18,
                    fontWeight: 600,
                    mb: 0.5,
                  }}
                >
                  Report
                </Typography>
                {sortedBlocks.map((block, index, sorted) => (
                  <BlockWithActions
                    key={block.id}
                    block={block}
                    index={index}
                    total={sorted.length}
                    onMoveUp={() =>
                      index > 0 && moveBlock(block, sorted[index - 1])
                    }
                    onMoveDown={() =>
                      index < sorted.length - 1 &&
                      moveBlock(block, sorted[index + 1])
                    }
                    onDelete={() => deleteBlock.mutate(block.id)}
                    isPending={patchBlock.isPending || deleteBlock.isPending}
                  />
                ))}
              </Box>
            )}

            <InvestigationChat
              messages={messages}
              value={chatText}
              onChange={setChatText}
              onSend={handleSendChat}
              sending={postChat.isPending}
            />

            <InvestigationComments
              comments={comments}
              value={commentText}
              onChange={setCommentText}
              onAdd={handleAddComment}
              adding={postComment.isPending}
            />
          </Stack>

          <Stack
            component="aside"
            aria-label="Investigation details"
            gap={2}
            sx={{ flex: '1 1 300px', minWidth: 0, maxWidth: { lg: 340 } }}
          >
            <Box sx={railCardSx}>
              <Typography component="h2" sx={railTitleSx}>
                Share and export
              </Typography>
              <Stack sx={{ mx: -1 }}>
                <Button
                  startIcon={<ContentCopyIcon />}
                  onClick={handleCopyLink}
                  sx={railActionSx}
                >
                  {copyDone ? 'Copied!' : 'Copy link'}
                </Button>
                <Button
                  startIcon={<PictureAsPdfIcon />}
                  onClick={() =>
                    window.open(
                      getInvestigationExportPdfUrl(inv.id),
                      '_blank',
                      'noopener,noreferrer'
                    )
                  }
                  sx={railActionSx}
                >
                  Export PDF
                </Button>
                <Button
                  startIcon={<ContentCopyIcon />}
                  onClick={() => void handleCopyMarkdown()}
                  sx={railActionSx}
                >
                  Copy markdown
                </Button>
                <Button
                  startIcon={<DataObjectIcon />}
                  onClick={() => void handleCopyEvidenceJson()}
                  sx={railActionSx}
                >
                  Copy JSON evidence
                </Button>
              </Stack>
            </Box>

            <Box sx={railCardSx}>
              <Typography component="h2" sx={railTitleSx}>
                Details
              </Typography>
              <Select
                size="small"
                fullWidth
                value={
                  STATUS_OPTIONS.includes(
                    inv.status as (typeof STATUS_OPTIONS)[number]
                  )
                    ? inv.status
                    : 'open'
                }
                onChange={(e) =>
                  patchInv.mutate({ status: e.target.value as string })
                }
                disabled={patchInv.isPending || isRunning}
                inputProps={{ 'aria-label': 'Status' }}
                sx={{
                  mb: 1.75,
                  fontSize: 14,
                  color: adreColors.text,
                  bgcolor: adreColors.card,
                  '& fieldset': { borderColor: adreColors.border },
                }}
              >
                {STATUS_OPTIONS.map((s) => (
                  <MenuItem key={s} value={s}>
                    {s.replace('_', ' ')}
                  </MenuItem>
                ))}
              </Select>
              <Box
                component="dl"
                sx={{
                  m: 0,
                  display: 'grid',
                  gridTemplateColumns: '84px minmax(0, 1fr)',
                  rowGap: 0.75,
                  fontSize: 13,
                  '& dt': { color: adreColors.muted },
                  '& dd': {
                    m: 0,
                    color: adreColors.body,
                    overflowWrap: 'anywhere',
                  },
                }}
              >
                {details.map(([label, value]) => (
                  <Fragment key={label}>
                    <dt>{label}</dt>
                    <dd>{value}</dd>
                  </Fragment>
                ))}
              </Box>
            </Box>

            <Box sx={railCardSx}>
              <Typography component="h2" sx={railTitleSx}>
                AI usage
              </Typography>
              {showUsageLoading ? (
                <Stack direction="row" gap={1} alignItems="center">
                  <CircularProgress size={14} />
                  <Typography sx={{ fontSize: 13, color: adreColors.muted }}>
                    Loading usage…
                  </Typography>
                </Stack>
              ) : usageSummary.hasUsage ? (
                <Stack gap={1}>
                  <Typography sx={{ fontSize: 14 }}>
                    {usageSummary.callCount}{' '}
                    {usageSummary.callCount === 1 ? 'call' : 'calls'} ·{' '}
                    {formatTokensWithCached(
                      usageSummary.totalTokens,
                      usageSummary.totalCached > 0
                        ? usageSummary.totalCached
                        : undefined
                    )}{' '}
                    · {formatUsdCost(usageSummary.totalCost)}
                  </Typography>
                  {usageSummary.steps.map((step) => (
                    <Box
                      key={step.id}
                      sx={{ fontSize: 12, color: adreColors.secondary }}
                    >
                      <Box component="span" sx={{ color: adreColors.body }}>
                        {HOLMES_FEATURE_LABELS[step.feature] ?? step.feature}
                      </Box>
                      {` · ${step.model || 'default'} · ${formatTokensWithCached(step.totalTokens, step.cachedTokens)} · ${formatUsdCost(step.totalCost)}`}
                      <Box sx={{ color: adreColors.muted }}>
                        {step.createdAt
                          ? new Date(step.createdAt).toLocaleString()
                          : '—'}
                      </Box>
                    </Box>
                  ))}
                </Stack>
              ) : (
                <Typography sx={{ fontSize: 13, color: adreColors.muted }}>
                  {isRunning
                    ? 'Usage shows here as the AI calls finish.'
                    : 'No AI usage yet.'}
                </Typography>
              )}
            </Box>
          </Stack>
        </Box>
        <Snackbar
          open={snackMessage != null}
          autoHideDuration={6000}
          onClose={() => setSnackMessage(null)}
          message={snackMessage ?? ''}
          ContentProps={{
            sx: {
              bgcolor:
                snackSeverity === 'error' ? 'error.main' : 'success.main',
              color: 'white',
            },
          }}
        />
      </Box>
    </Page>
  );
};

export default InvestigationDetailPage;
