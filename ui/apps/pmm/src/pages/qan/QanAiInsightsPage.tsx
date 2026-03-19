import {
  Alert,
  Button,
  Card,
  CardContent,
  CircularProgress,
  Stack,
  TextField,
  Typography,
} from '@mui/material';
import RefreshIcon from '@mui/icons-material/Refresh';
import { FC, useEffect, useRef, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import Markdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import rehypeRaw from 'rehype-raw';
import { Page } from 'components/page';
import { adreQanInsights, getQanInsightsCache } from 'api/adre';
import { getMarkdownComponents } from 'components/adre/adre-chat-markdown';

const RUNNING_MESSAGE =
  'Query analysis and optimisation is running. Results will appear here soon.';

function getParam(params: URLSearchParams, snakeKey: string): string {
  const camelKey = snakeKey.replace(/_([a-z])/g, (_, c) => c.toUpperCase());
  return params.get(snakeKey) ?? params.get(camelKey) ?? '';
}

function formatCacheTimestamp(iso: string): string {
  try {
    const d = new Date(iso);
    if (Number.isNaN(d.getTime())) return iso;
    return d.toLocaleString(undefined, {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  } catch {
    return iso;
  }
}

const QanAiInsightsPage: FC = () => {
  const [searchParams] = useSearchParams();
  const [analysis, setAnalysis] = useState<string | null>(null);
  const [cachedAt, setCachedAt] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [manualServiceId, setManualServiceId] = useState('');
  const [manualQueryText, setManualQueryText] = useState('');
  const runOnceRef = useRef(false);
  const mountedRef = useRef(true);

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
    };
  }, []);

  const urlServiceId = getParam(searchParams, 'service_id');
  const urlQueryText = getParam(searchParams, 'query_text');
  const hasUrlContext = Boolean(urlServiceId.trim() && urlQueryText.trim());

  const serviceId = urlServiceId || manualServiceId;
  const queryText = urlQueryText || manualQueryText;
  const queryId = getParam(searchParams, 'query_id');
  const fingerprint = getParam(searchParams, 'fingerprint');
  const timeFrom = getParam(searchParams, 'from');
  const timeTo = getParam(searchParams, 'to');

  const hasContext = Boolean(serviceId.trim() && queryText.trim());

  const runAnalysis = (force: boolean) => {
    setLoading(true);
    setError(null);
    adreQanInsights({
      serviceId: serviceId.trim(),
      queryText: queryText.trim(),
      ...(queryId && { queryId }),
      ...(fingerprint && { fingerprint }),
      ...(timeFrom && { timeFrom }),
      ...(timeTo && { timeTo }),
      force,
    })
      .then((res) => {
        if (mountedRef.current) {
          setAnalysis(res.analysis ?? '');
          setCachedAt(res.created_at ?? null);
        }
      })
      .catch((err: Error & { response?: { data?: { error?: string } } }) => {
        if (mountedRef.current) {
          setError(err?.response?.data?.error ?? err?.message ?? 'Failed to get AI insights');
        }
      })
      .finally(() => {
        if (mountedRef.current) setLoading(false);
      });
  };

  useEffect(() => {
    if (!hasUrlContext || runOnceRef.current || loading || analysis) return;
    runOnceRef.current = true;

    if (queryId) {
      setLoading(true);
      getQanInsightsCache(queryId, urlServiceId.trim())
        .then((cached) => {
          if (!mountedRef.current) return;
          if (cached?.analysis) {
            setAnalysis(cached.analysis);
            setCachedAt(cached.created_at ?? null);
            setLoading(false);
          } else {
            runAnalysis(false);
          }
        })
        .catch(() => {
          if (mountedRef.current) runAnalysis(false);
        });
    } else {
      runAnalysis(false);
    }
  }, [hasUrlContext, urlServiceId, urlQueryText, queryId, fingerprint, timeFrom, timeTo]);

  const handleRunManual = () => {
    if (!manualServiceId.trim() || !manualQueryText.trim()) return;
    runOnceRef.current = false;
    setAnalysis(null);
    setCachedAt(null);
    setError(null);
    runAnalysis(false);
  };

  return (
    <Page title="QAN AI Insights">
      <Stack gap={2} sx={{ maxWidth: 900 }}>
        {hasContext && (
          <Typography variant="body2" color="text.secondary">
            {hasUrlContext ? 'Opened from Query Analytics. ' : ''}
            Service: {serviceId}
            {queryId && ` · Query ID: ${queryId}`}
          </Typography>
        )}
        {hasContext && (
          <Card variant="outlined">
            <CardContent>
              <Typography variant="subtitle2" color="text.secondary" gutterBottom>
                Query
              </Typography>
              <Typography
                component="pre"
                variant="body2"
                sx={{
                  whiteSpace: 'pre-wrap',
                  wordBreak: 'break-word',
                  maxHeight: 120,
                  overflow: 'auto',
                  fontFamily: 'monospace',
                }}
              >
                {queryText.length > 500 ? `${queryText.slice(0, 500)}…` : queryText}
              </Typography>
            </CardContent>
          </Card>
        )}

        {loading && (
          <Card variant="outlined">
            <CardContent>
              <Stack direction="row" alignItems="center" gap={2}>
                <CircularProgress size={24} />
                <Typography variant="body2" color="text.secondary">
                  {RUNNING_MESSAGE}
                </Typography>
              </Stack>
            </CardContent>
          </Card>
        )}

        {error && (
          <Alert severity="error" onClose={() => setError(null)}>
            {error}
          </Alert>
        )}

        {!loading && analysis !== null && (
          <Card variant="outlined">
            <CardContent>
              <Stack direction="row" alignItems="center" justifyContent="space-between" sx={{ mb: 1 }}>
                <Stack>
                  <Typography variant="subtitle1" fontWeight={600}>
                    Analysis
                  </Typography>
                  {cachedAt && (
                    <Typography variant="caption" color="text.secondary">
                      Last analyzed: {formatCacheTimestamp(cachedAt)}
                    </Typography>
                  )}
                </Stack>
                <Button
                  variant="outlined"
                  size="small"
                  startIcon={<RefreshIcon />}
                  onClick={() => runAnalysis(true)}
                  disabled={loading}
                >
                  Re-run Analysis
                </Button>
              </Stack>
              <Typography
                component="div"
                variant="body2"
                sx={{ '& p': { mb: 1 }, '& pre': { overflow: 'auto' } }}
              >
                <Markdown
                  remarkPlugins={[remarkGfm]}
                  rehypePlugins={[rehypeRaw]}
                  components={getMarkdownComponents(analysis ?? '')}
                >
                  {analysis}
                </Markdown>
              </Typography>
            </CardContent>
          </Card>
        )}

        {!hasUrlContext && !loading && (
          <>
            <Typography variant="body2" color="text.secondary">
              Select a query in Query Analytics and use the AI Insights button there to
              analyze it. Or enter the service and query below.
            </Typography>
            <Card variant="outlined">
              <CardContent>
                <Stack gap={2}>
                  <TextField
                    label="Service ID"
                    value={manualServiceId}
                    onChange={(e) => setManualServiceId(e.target.value)}
                    placeholder="e.g. service-id-from-inventory"
                    size="small"
                    fullWidth
                  />
                  <TextField
                    label="Query text"
                    value={manualQueryText}
                    onChange={(e) => setManualQueryText(e.target.value)}
                    placeholder="SQL or query fingerprint"
                    size="small"
                    fullWidth
                    multiline
                    minRows={3}
                  />
                  <Button
                    variant="contained"
                    onClick={handleRunManual}
                    disabled={
                      !manualServiceId.trim() ||
                      !manualQueryText.trim() ||
                      loading
                    }
                  >
                    Run analysis
                  </Button>
                </Stack>
              </CardContent>
            </Card>
          </>
        )}
      </Stack>
    </Page>
  );
};

export default QanAiInsightsPage;
