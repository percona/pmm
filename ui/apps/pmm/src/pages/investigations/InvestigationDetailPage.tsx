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
  Typography,
} from '@mui/material';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import PictureAsPdfIcon from '@mui/icons-material/PictureAsPdf';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import ArrowUpwardIcon from '@mui/icons-material/ArrowUpward';
import ArrowDownwardIcon from '@mui/icons-material/ArrowDownward';
import { FC, useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
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
} from 'hooks/api/useInvestigations';
import { PMM_NEW_NAV_PATH } from 'lib/constants';
import { getInvestigationExportPdfUrl } from 'api/investigations';
import type { InvestigationBlock } from 'api/investigations';
import {
  getAdreAlerts,
  getAlertMetadataFromLabels,
  type AlertMetadataFromLabels,
} from 'api/adre';
import { BlockRenderer } from './components/BlockRenderer';
import { TimelineSection } from './components/TimelineSection';

const STATUS_OPTIONS = ['open', 'in_progress', 'investigating', 'resolved', 'archived'] as const;

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
  const { data: inv, isLoading, isError, error } = useInvestigation(id);
  const { data: comments = [] } = useInvestigationComments(id);
  const { data: messages = [] } = useInvestigationMessages(id, { limit: 50 });
  const { data: timelineEvents = [] } = useInvestigationTimeline(id);
  const postComment = usePostInvestigationComment(id ?? '');
  const postChat = usePostInvestigationChat(id ?? '');
  const postRun = usePostInvestigationRun(id ?? '');
  const patchInv = usePatchInvestigation(id ?? '');
  const patchBlock = usePatchInvestigationBlock(id ?? '');
  const deleteBlock = useDeleteInvestigationBlock(id ?? '');
  const [commentText, setCommentText] = useState('');
  const [chatText, setChatText] = useState('');
  const [copyDone, setCopyDone] = useState(false);
  const [snackMessage, setSnackMessage] = useState<string | null>(null);
  const [snackSeverity, setSnackSeverity] = useState<'error' | 'success'>('error');
  const [fetchedAlertMeta, setFetchedAlertMeta] = useState<AlertMetadataFromLabels>({});

  // When investigation is from an alert but API didn't return node/service, fetch alerts and derive metadata
  useEffect(() => {
    if (!inv) return;
    setFetchedAlertMeta({});
    if (inv.sourceType !== 'alert' || !inv.sourceRef) return;
    const refs = new Set(inv.sourceRef.split(',').map((s) => s.trim()).filter(Boolean));
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
  }, [inv?.id, inv?.sourceType, inv?.sourceRef]);

  const showError = (msg: string) => {
    setSnackMessage(msg);
    setSnackSeverity('error');
  };
  const showSuccess = (msg: string) => {
    setSnackMessage(msg);
    setSnackSeverity('success');
  };
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

  const timeFrom = inv.timeFrom ?? (inv as { time_from?: string }).time_from;
  const timeTo = inv.timeTo ?? (inv as { time_to?: string }).time_to;
  const timeRange =
    timeFrom && timeTo
      ? `${new Date(timeFrom).toLocaleString()} — ${new Date(timeTo).toLocaleString()}`
      : null;

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
              disabled={patchInv.isPending}
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
          <Button
            variant="contained"
            size="small"
            onClick={() =>
              id &&
              postRun.mutate(undefined, {
                onError: (err) => showError(`Run failed: ${getErrorMessage(err)}`),
                onSuccess: () => showSuccess('Investigation run completed'),
              })
            }
            disabled={postRun.isPending}
          >
            {postRun.isPending ? 'Running…' : 'Run investigation'}
          </Button>
        </Stack>
      }
    >
      {/* Summary */}
      {inv.summary && (
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

      {/* Detailed summary */}
      {(inv.summaryDetailed || inv.rootCauseSummary || inv.resolutionSummary) && (
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
    </Page>
  );
};

export default InvestigationDetailPage;
