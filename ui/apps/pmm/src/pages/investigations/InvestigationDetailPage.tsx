import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  CircularProgress,
  Divider,
  IconButton,
  Stack,
  TextField,
  Typography,
} from '@mui/material';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import { FC, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Page } from 'components/page';
import {
  useInvestigation,
  useInvestigationComments,
  usePostInvestigationComment,
} from 'hooks/api/useInvestigations';
import { PMM_NEW_NAV_PATH } from 'lib/constants';
import { BlockRenderer } from './components/BlockRenderer';

const InvestigationDetailPage: FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { data: inv, isLoading, isError, error } = useInvestigation(id);
  const { data: comments = [] } = useInvestigationComments(id);
  const postComment = usePostInvestigationComment(id ?? '');
  const [commentText, setCommentText] = useState('');
  const [copyDone, setCopyDone] = useState(false);

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

  const timeRange =
    inv.timeFrom && inv.timeTo
      ? `${new Date(inv.timeFrom).toLocaleString()} — ${new Date(inv.timeTo).toLocaleString()}`
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
          <Chip label={inv.status} size="small" variant="outlined" />
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
        </Stack>
      }
    >
      {/* Short summary */}
      {inv.summary && (
        <Card variant="outlined" sx={{ mb: 2 }}>
          <CardContent>
            <Typography variant="subtitle2" color="text.secondary" gutterBottom>
              Summary
            </Typography>
            <Typography variant="body1" sx={{ whiteSpace: 'pre-wrap' }}>
              {inv.summary}
            </Typography>
          </CardContent>
        </Card>
      )}

      {/* Metadata */}
      <Stack direction="row" flexWrap="wrap" gap={2} sx={{ mb: 2 }}>
        {timeRange && (
          <Typography variant="body2" color="text.secondary">
            Time range: {timeRange}
          </Typography>
        )}
        {inv.sourceType && (
          <Typography variant="body2" color="text.secondary">
            Source: {inv.sourceType}
          </Typography>
        )}
      </Stack>

      {/* Report body: blocks */}
      {inv.blocks && inv.blocks.length > 0 && (
        <>
          <Typography variant="h6" sx={{ mt: 3, mb: 1 }}>
            Report
          </Typography>
          {inv.blocks
            .sort((a, b) => a.position - b.position)
            .map((block) => (
              <BlockRenderer key={block.id} block={block} />
            ))}
        </>
      )}

      {/* Detailed summary */}
      {(inv.summaryDetailed || inv.rootCauseSummary || inv.resolutionSummary) && (
        <>
          <Typography variant="h6" sx={{ mt: 3, mb: 1 }}>
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
    </Page>
  );
};

export default InvestigationDetailPage;
