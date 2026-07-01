import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  CircularProgress,
  Divider,
  FormControl,
  IconButton,
  MenuItem,
  Select,
  Stack,
  Snackbar,
  TextField,
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
import { FC, useState, useEffect, useRef } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useQueryClient } from '@tanstack/react-query';
import { Page } from 'components/page';
import {
  useInvestigation,
  useInvestigationComments,
  useInvestigationMessages,
  useInvestigationTimeline,
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
import { HolmesUsageFooter } from 'components/adre/HolmesUsageFooter';
import {
  aggregateInvestigationUsage,
  formatTokensWithCached,
  formatUsdCost,
  HOLMES_FEATURE_LABELS,
} from 'utils/holmesUsageFormat';
import { BlockRenderer } from './components/BlockRenderer';
import { TimelineSection } from './components/TimelineSection';

const STATUS_OPTIONS = ['open', 'in_progress', 'investigating', 'running', 'completed', 'failed', 'resolved', 'archived'] as const;

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
  <Box sx={{ display: 'flex', alignItems: 'flex-start', gap: 0.5, mb: 2 }}>
    <Box sx={{ flex: 1, minWidth: 0 }}>
      <BlockRenderer block={block} />
    </Box>
    <Stack direction="row" sx={{ mt: 1 }} spacing={0}>
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
        color="error"
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
  const { data: inv, isLoading, isError, error } = useInvestigation(id, {
    refetchInterval: isRunning ? 5000 : false,
  });
  const { data: comments = [] } = useInvestigationComments(id);
  const { data: messages = [] } = useInvestigationMessages(
    id,
    { limit: 50 },
    { refetchInterval: isRunning ? 5000 : false }
  );
  const { data: timelineEvents = [] } = useInvestigationTimeline(id);
  const postComment = usePostInvestigationComment(id ?? '');
  const postChat = usePostInvestigationChat(id ?? '');
  const postRun = usePostInvestigationRun(id ?? '');
  const patchInv = usePatchInvestigation(id ?? '');
  const patchBlock = usePatchInvestigationBlock(id ?? '');
  const deleteBlock = useDeleteInvestigationBlock(id ?? '');
  const createSNTicket = useCreateServiceNowTicket(id ?? '');
  const { data: adreSettings } = useAdreSettings();
  const { data: usageBreakdown, isLoading: usageLoading } = useInvestigationUsage(id, {
    refetchInterval: isRunning ? 5000 : false,
  });
  const [commentText, setCommentText] = useState('');
  const [chatText, setChatText] = useState('');
  const [copyDone, setCopyDone] = useState(false);
  const [snackMessage, setSnackMessage] = useState<string | null>(null);
  const [snackSeverity, setSnackSeverity] = useState<'error' | 'success'>('error');
  const [fetchedAlertMeta, setFetchedAlertMeta] = useState<AlertMetadataFromLabels>({});
  const [showEvidence, setShowEvidence] = useState(false);
  const prevStatusRef = useRef<string | undefined>();

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
    if (prev === 'running' && (status === 'completed' || status === 'failed') && id) {
      void queryClient.invalidateQueries({ queryKey: INVESTIGATIONS_KEYS.detail(id) });
      void queryClient.invalidateQueries({ queryKey: INVESTIGATIONS_KEYS.messagesPrefix(id) });
      void queryClient.invalidateQueries({ queryKey: INVESTIGATIONS_KEYS.usage(id) });
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
    const refs = new Set(invSourceRef.split(',').map((s) => s.trim()).filter(Boolean));
    if (refs.size === 0) return;
    let cancelled = false;
    getAdreAlerts()
      .then((data: unknown) => {
        if (cancelled) return;
        const raw = data as {
          data?: { alerts?: Array<{ fingerprint?: string; labels?: Record<string, string> }> };
          alerts?: Array<{ fingerprint?: string; labels?: Record<string, string> }>;
        };
        const list = raw?.data?.alerts ?? raw?.alerts ?? [];
        const arr = Array.isArray(list) ? list : [];
        const match = arr.find(
          (a) => a.fingerprint && refs.has(a.fingerprint)
        );
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
    return ax?.response?.data?.error ?? (err as Error)?.message ?? 'Request failed';
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
    const blocks = [...(inv.blocks ?? [])].sort((a, b) => a.position - b.position);
    const blockText = blocks
      .map((b) => {
        const content = (b.dataJson as { content?: string; steps?: string[] } | undefined);
        if (content?.content) return `## ${b.title || b.type}\n${content.content}`;
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
              {inv ? 'Failed to load investigation.' : 'Investigation not found.'}
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
      ? `${new Date(timeFrom).toLocaleString()} — ${new Date(timeTo).toLocaleString()}`
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

  return (
    <Page
      title={inv.title || 'Investigation'}
      topBar={
        <Stack direction="row" alignItems="center" gap={1} sx={{ mb: 1 }}>
          <IconButton
            size="small"
            onClick={() => navigate(`${PMM_NEW_NAV_PATH}/investigations`)}
            aria-label="Back to list"
          >
            <ArrowBackIcon />
          </IconButton>
          <FormControl size="small" sx={{ minWidth: 140 }}>
            <Select
              value={STATUS_OPTIONS.includes(inv.status as (typeof STATUS_OPTIONS)[number]) ? inv.status : 'open'}
              onChange={(e) =>
                patchInv.mutate({ status: e.target.value as string })
              }
              displayEmpty
              disabled={patchInv.isPending || isRunning}
            >
              {STATUS_OPTIONS.map((s) => (
                <MenuItem key={s} value={s}>
                  {s.replace('_', ' ')}
                </MenuItem>
              ))}
            </Select>
          </FormControl>
          {inv.severity && (
            <Chip label={inv.severity} size="small" variant="outlined" />
          )}
          <Box sx={{ flex: 1 }} />
          <Button
            size="small"
            startIcon={<ContentCopyIcon />}
            onClick={handleCopyLink}
          >
            {copyDone ? 'Copied!' : 'Copy link'}
          </Button>
          <Button
            size="small"
            startIcon={<PictureAsPdfIcon />}
            onClick={() => id && window.open(getInvestigationExportPdfUrl(id), '_blank', 'noopener,noreferrer')}
          >
            Export PDF
          </Button>
          <Button size="small" startIcon={<ContentCopyIcon />} onClick={() => void handleCopyMarkdown()}>
            Copy markdown
          </Button>
          <Button size="small" startIcon={<DataObjectIcon />} onClick={() => void handleCopyEvidenceJson()}>
            Copy JSON evidence
          </Button>
          {(() => {
            const ticketId = inv.servicenowTicketId ?? inv.servicenow_ticket_id;
            const ticketNumber = inv.servicenowTicketNumber ?? inv.servicenow_ticket_number;
            const snConfigured = adreSettings?.servicenowConfigured ?? adreSettings?.servicenow_configured ?? false;
            if (ticketId) {
              const snApiUrl = adreSettings?.servicenowUrl ?? adreSettings?.servicenow_url ?? '';
              let instanceUrl = '';
              try {
                const u = new URL(snApiUrl);
                instanceUrl = u.origin;
              } catch { /* ignore */ }
              const label = ticketNumber || ticketId;
              const href = instanceUrl ? `${instanceUrl}/nav_to.do?uri=incident.do?sys_id=${ticketId}` : '';
              return (
                <Chip
                  icon={<CheckCircleOutlineIcon />}
                  label={`ServiceNow: ${label}`}
                  color="success"
                  size="small"
                  variant="outlined"
                  clickable={!!href}
                  onClick={href ? () => window.open(href, '_blank', 'noopener,noreferrer') : undefined}
                />
              );
            }
            return (
              <Tooltip
                title={snConfigured ? '' : 'Configure ServiceNow in AI Assistant settings'}
              >
                <span>
                  <Button
                    size="small"
                    variant="outlined"
                    color="success"
                    disabled={!snConfigured || createSNTicket.isPending}
                    onClick={() =>
                      id &&
                      createSNTicket.mutate(undefined, {
                        onError: (err) => showError(`ServiceNow: ${getErrorMessage(err)}`),
                        onSuccess: (data) => showSuccess(`ServiceNow ticket created: ${data.ticket_number || data.ticket_id}`),
                      })
                    }
                  >
                    {createSNTicket.isPending ? 'Creating…' : 'Create ServiceNow Ticket'}
                  </Button>
                </span>
              </Tooltip>
            );
          })()}
          <Button
            variant="contained"
            size="small"
            onClick={() =>
              id &&
              postRun.mutate(undefined, {
                onError: (err) => showError(`Run failed: ${getErrorMessage(err)}`),
                onSuccess: () => setIsRunning(true),
              })
            }
            disabled={postRun.isPending || isRunning}
          >
            {isRunning ? 'Running…' : 'Run investigation'}
          </Button>
        </Stack>
      }
    >
      {/* flexShrink:0 keeps the Page's flex column from squishing these cards.
          MUI Card defaults to overflow:hidden, so a shrunk card clips its
          content instead of letting the page scroll. */}
      <Box sx={{ flexShrink: 0 }}>
      {/* Running banner */}
      {isRunning && (
        <Alert severity="info" icon={<CircularProgress size={20} />} sx={{ mb: 2 }}>
          Investigation is running. Results will appear automatically when complete.
        </Alert>
      )}
      {(() => {
        const userRequest = investigationUserRequest(inv);
        if (!userRequest) {
          return null;
        }
        return (
          <>
            <Typography variant="h6" sx={{ mb: 1 }}>
              Your request
            </Typography>
            <Card variant="outlined" sx={{ mb: 2 }}>
              <CardContent>
                <Typography variant="body1" sx={{ whiteSpace: 'pre-wrap' }}>
                  {userRequest}
                </Typography>
              </CardContent>
            </Card>
          </>
        );
      })()}
      {/* Summary (fallback only — superseded by the Summary report block when present) */}
      {inv.summary && !hasReportBlocks && (
        <>
          <Typography variant="h6" sx={{ mb: 1 }}>
            Summary
          </Typography>
          <Card variant="outlined" sx={{ mb: 2, bgcolor: 'action.hover' }}>
            <CardContent>
              <Typography variant="body1" sx={{ whiteSpace: 'pre-wrap' }}>
                {inv.summary}
              </Typography>
            </CardContent>
          </Card>
        </>
      )}
      <Card variant="outlined" sx={{ mb: 2 }}>
        <CardContent>
          <Stack direction="row" alignItems="center" justifyContent="space-between" sx={{ mb: 1 }}>
            <Typography variant="subtitle2">
              Confidence: {(inv.confidence || 'medium').toUpperCase()} ({inv.confidenceScore ?? 0})
            </Typography>
            <Button size="small" onClick={() => setShowEvidence((v) => !v)}>
              {showEvidence ? 'Hide evidence map' : `Show evidence map (${inv.evidence?.length ?? 0})`}
            </Button>
          </Stack>
          {!!inv.confidenceRationale && (
            <Typography variant="body2" color="text.secondary" sx={{ mb: showEvidence ? 1 : 0 }}>
              {inv.confidenceRationale}
            </Typography>
          )}
          {showEvidence &&
            (inv.evidence?.length ? (
              <Stack spacing={1}>
                {inv.evidence.map((e, idx) => (
                  <Card key={`${e.id || idx}`} variant="outlined">
                    <CardContent>
                      <Typography variant="subtitle2">{e.claim || 'Claim'}</Typography>
                      <Typography variant="caption" color="text.secondary">
                        {e.kind} · {e.source_tool} · {e.source_ref}
                      </Typography>
                      {!!e.excerpt && (
                        <Typography variant="body2" sx={{ mt: 0.5, whiteSpace: 'pre-wrap' }}>
                          {e.excerpt}
                        </Typography>
                      )}
                    </CardContent>
                  </Card>
                ))}
              </Stack>
            ) : (
              <Typography variant="body2" color="text.secondary">
                No evidence entries available.
              </Typography>
            ))}
        </CardContent>
      </Card>

      {/* Metadata row */}
      <Stack direction="row" flexWrap="wrap" gap={2} sx={{ mb: 2 }}>
        {timeRange && (
          <Typography variant="body2" color="text.secondary">
            Time range: {timeRange}
          </Typography>
        )}
        {inv.sourceType && (
          <Typography variant="body2" color="text.secondary">
            Source:{' '}
            {inv.sourceType === 'alert' ? 'Alert' : 'User request'}
          </Typography>
        )}
        {(inv.nodeName ?? (inv as { node_name?: string }).node_name ?? fetchedAlertMeta.nodeName) && (
          <Typography variant="body2" color="text.secondary">
            Node: {inv.nodeName ?? (inv as { node_name?: string }).node_name ?? fetchedAlertMeta.nodeName}
          </Typography>
        )}
        {(inv.serviceName ?? (inv as { service_name?: string }).service_name ?? fetchedAlertMeta.serviceName) && (
          <Typography variant="body2" color="text.secondary">
            Service: {inv.serviceName ?? (inv as { service_name?: string }).service_name ?? fetchedAlertMeta.serviceName}
          </Typography>
        )}
        {(inv.clusterName ?? (inv as { cluster_name?: string }).cluster_name ?? fetchedAlertMeta.clusterName) && (
          <Typography variant="body2" color="text.secondary">
            Cluster: {inv.clusterName ?? (inv as { cluster_name?: string }).cluster_name ?? fetchedAlertMeta.clusterName}
          </Typography>
        )}
        {(inv.severity ?? (inv as { severity?: string }).severity ?? fetchedAlertMeta.severity) && (
          <Typography variant="body2" color="text.secondary">
            Severity: {inv.severity ?? (inv as { severity?: string }).severity ?? fetchedAlertMeta.severity}
          </Typography>
        )}
      </Stack>

      {/* Timeline */}
      <TimelineSection events={timelineEvents} />

      {/* Report body: blocks */}
      {inv.blocks && inv.blocks.length > 0 && (
        <>
          <Typography variant="h6" sx={{ mb: 1 }}>
            Report
          </Typography>
          {[...inv.blocks]
            .sort((a, b) => a.position - b.position)
            .map((block, index, sorted) => (
              <BlockWithActions
                key={block.id}
                block={block}
                index={index}
                total={sorted.length}
                onMoveUp={() => {
                  if (index <= 0) return;
                  const prev = sorted[index - 1];
                  patchBlock.mutate(
                    { blockId: block.id, body: { position: prev.position } },
                    {
                      onSuccess: () =>
                        patchBlock.mutate({
                          blockId: prev.id,
                          body: { position: block.position },
                        }),
                    }
                  );
                }}
                onMoveDown={() => {
                  if (index >= sorted.length - 1) return;
                  const next = sorted[index + 1];
                  patchBlock.mutate(
                    { blockId: block.id, body: { position: next.position } },
                    {
                      onSuccess: () =>
                        patchBlock.mutate({
                          blockId: next.id,
                          body: { position: block.position },
                        }),
                    }
                  );
                }}
                onDelete={() => deleteBlock.mutate(block.id)}
                isPending={
                  patchBlock.isPending || deleteBlock.isPending
                }
              />
            ))}
        </>
      )}

      {/* Detailed summary (fallback only — superseded by the report blocks when present) */}
      {!hasReportBlocks && (inv.summaryDetailed || inv.rootCauseSummary || inv.resolutionSummary) && (
        <>
          <Typography variant="h6" sx={{ mt: 2, mb: 1 }}>
            Detailed summary
          </Typography>
          <Card variant="outlined" sx={{ mb: 2 }}>
            <CardContent>
              {inv.summaryDetailed && (
                <Typography variant="body2" sx={{ whiteSpace: 'pre-wrap', mb: 1 }}>
                  {inv.summaryDetailed}
                </Typography>
              )}
              {inv.rootCauseSummary && (
                <>
                  <Typography variant="subtitle2" color="text.secondary">
                    Root cause
                  </Typography>
                  <Typography variant="body2" sx={{ whiteSpace: 'pre-wrap', mb: 1 }}>
                    {inv.rootCauseSummary}
                  </Typography>
                </>
              )}
              {inv.resolutionSummary && (
                <>
                  <Typography variant="subtitle2" color="text.secondary">
                    Resolution
                  </Typography>
                  <Typography variant="body2" sx={{ whiteSpace: 'pre-wrap' }}>
                    {inv.resolutionSummary}
                  </Typography>
                </>
              )}
            </CardContent>
          </Card>
        </>
      )}

      <Typography variant="h6" sx={{ mt: 2, mb: 1 }}>
        Usage details
      </Typography>
      <Card variant="outlined" sx={{ mb: 2 }}>
        <CardContent sx={{ py: 2 }}>
          {showUsageLoading ? (
            <Stack direction="row" spacing={1} alignItems="center">
              <CircularProgress size={16} />
              <Typography variant="body2" color="text.secondary">
                Loading usage…
              </Typography>
            </Stack>
          ) : usageSummary.hasUsage ? (
            <>
              <Typography variant="body2" sx={{ mb: usageSummary.steps.length > 0 ? 1.5 : 0 }}>
                {usageSummary.callCount} {usageSummary.callCount === 1 ? 'call' : 'calls'} ·{' '}
                {formatTokensWithCached(
                  usageSummary.totalTokens,
                  usageSummary.totalCached > 0 ? usageSummary.totalCached : undefined
                )}{' '}
                · {formatUsdCost(usageSummary.totalCost)}
              </Typography>
              {usageSummary.steps.length > 0 ? (
                <Stack spacing={0.75}>
                  {usageSummary.steps.map((step) => (
                    <Stack
                      key={step.id}
                      direction={{ xs: 'column', sm: 'row' }}
                      spacing={{ xs: 0.25, sm: 2 }}
                      sx={{ py: 0.25 }}
                    >
                      <Typography variant="caption" color="text.secondary" sx={{ minWidth: 140 }}>
                        {step.createdAt ? new Date(step.createdAt).toLocaleString() : '—'}
                      </Typography>
                      <Typography variant="caption" color="text.secondary" sx={{ minWidth: 120 }}>
                        {HOLMES_FEATURE_LABELS[step.feature] ?? step.feature}
                      </Typography>
                      <Typography variant="caption" color="text.secondary" sx={{ minWidth: 80 }}>
                        {step.model || 'default'}
                      </Typography>
                      <Typography variant="caption" color="text.secondary">
                        {formatTokensWithCached(step.totalTokens, step.cachedTokens)}{' '}
                        · {formatUsdCost(step.totalCost)}
                      </Typography>
                    </Stack>
                  ))}
                </Stack>
              ) : (
                <Typography variant="body2" color="text.secondary">
                  {isRunning
                    ? 'Investigation in progress — per-step breakdown will appear as AI calls complete.'
                    : 'Per-step breakdown will appear after AI calls complete.'}
                </Typography>
              )}
            </>
          ) : (
            <Typography variant="body2" color="text.secondary">
              {isRunning
                ? 'Investigation in progress — usage will appear when AI calls complete.'
                : 'No AI usage recorded yet. Run the investigation or send a chat message to generate usage data.'}
            </Typography>
          )}
        </CardContent>
      </Card>

      <Divider sx={{ my: 3 }} />

      {/* Comments */}
      <Typography variant="h6" sx={{ mb: 2 }}>
        Comments
      </Typography>
      <Stack spacing={2}>
        {comments.map((c) => (
          <Card key={c.id} variant="outlined">
            <CardContent>
              <Typography variant="caption" color="text.secondary">
                {c.author || 'Anonymous'} ·{' '}
                {new Date(c.createdAt).toLocaleString()}
              </Typography>
              <Typography variant="body2" sx={{ mt: 0.5, whiteSpace: 'pre-wrap' }}>
                {c.content}
              </Typography>
            </CardContent>
          </Card>
        ))}
        <Card variant="outlined">
          <CardContent>
            <TextField
              fullWidth
              multiline
              minRows={2}
              placeholder="Add a comment..."
              value={commentText}
              onChange={(e) => setCommentText(e.target.value)}
              size="small"
              onKeyDown={(e) => {
                if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
                  e.preventDefault();
                  handleAddComment();
                }
              }}
            />
            <Button
              variant="contained"
              size="small"
              sx={{ mt: 1 }}
              onClick={handleAddComment}
              disabled={!commentText.trim() || postComment.isPending}
            >
              Add comment
            </Button>
          </CardContent>
        </Card>
      </Stack>

      <Divider sx={{ my: 3 }} />

      <Typography variant="h6" sx={{ mb: 2 }}>
        Chat
      </Typography>
      <Stack spacing={2}>
        {messages.length === 0 ? (
          <Typography variant="body2" color="text.secondary">
            No messages yet. Ask a question about this investigation.
          </Typography>
        ) : (
          [...messages]
            .filter((m) => m.role === 'user' || m.role === 'assistant')
            .reverse()
            .map((m) => (
              <Card
                key={m.id}
                variant="outlined"
                sx={{
                  alignSelf: m.role === 'user' ? 'flex-end' : 'flex-start',
                  maxWidth: '85%',
                  bgcolor: m.role === 'user' ? 'action.hover' : undefined,
                }}
              >
                <CardContent>
                  <Typography variant="caption" color="text.secondary">
                    {m.role === 'user' ? 'You' : 'Assistant'}
                    {' · '}
                    {new Date(m.createdAt).toLocaleString()}
                  </Typography>
                  <Typography variant="body2" sx={{ mt: 0.5, whiteSpace: 'pre-wrap' }}>
                    {m.content}
                  </Typography>
                  {m.role === 'assistant' ? (
                    <HolmesUsageFooter
                      usage={{
                        model: m.model,
                        promptTokens: m.promptTokens ?? m.prompt_tokens,
                        completionTokens: m.completionTokens ?? m.completion_tokens,
                        totalTokens: m.totalTokens ?? m.total_tokens,
                        cachedTokens: m.cachedTokens ?? m.cached_tokens,
                        totalCost: m.totalCost ?? m.total_cost,
                      }}
                      align="left"
                    />
                  ) : null}
                </CardContent>
              </Card>
            ))
        )}
        <Card variant="outlined">
          <CardContent>
            <TextField
              fullWidth
              multiline
              minRows={2}
              placeholder="Ask about this investigation..."
              value={chatText}
              onChange={(e) => setChatText(e.target.value)}
              size="small"
              onKeyDown={(e) => {
                if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
                  e.preventDefault();
                  handleSendChat();
                }
              }}
            />
            <Button
              variant="contained"
              size="small"
              sx={{ mt: 1 }}
              onClick={handleSendChat}
              disabled={!chatText.trim() || postChat.isPending}
            >
              {postChat.isPending ? 'Sending…' : 'Send'}
            </Button>
          </CardContent>
        </Card>
      </Stack>
      <Snackbar
        open={snackMessage != null}
        autoHideDuration={6000}
        onClose={() => setSnackMessage(null)}
        message={snackMessage ?? ''}
        ContentProps={{
          sx: { bgcolor: snackSeverity === 'error' ? 'error.main' : 'success.main', color: 'white' },
        }}
      />
      </Box>
    </Page>
  );
};

export default InvestigationDetailPage;
