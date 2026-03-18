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
import { FC, useEffect, useRef, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import Markdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { Page } from 'components/page';
import { adreQanInsights } from 'api/adre';
import { CodeBlock } from 'pages/updates/change-log/code-block';

const RUNNING_MESSAGE =
  'Query analysis and optimisation is running. Results will appear here soon.';

function getParam(params: URLSearchParams, snakeKey: string): string {
  const camelKey = snakeKey.replace(/_([a-z])/g, (_, c) => c.toUpperCase());
  return params.get(snakeKey) ?? params.get(camelKey) ?? '';
}

const QanAiInsightsPage: FC = () => {
  const [searchParams] = useSearchParams();
  const [analysis, setAnalysis] = useState<string | null>(null);
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

  useEffect(() => {
    if (!hasUrlContext || runOnceRef.current || loading || analysis) return;
    runOnceRef.current = true;
    setLoading(true);
    setError(null);
    let cancelled = false;
    const checkMounted = () => !cancelled && mountedRef.current;
    adreQanInsights({
      serviceId: urlServiceId.trim(),
      queryText: urlQueryText.trim(),
      ...(queryId && { queryId }),
      ...(fingerprint && { fingerprint }),
      ...(timeFrom && { timeFrom }),
      ...(timeTo && { timeTo }),
    })
      .then((res) => {
        if (checkMounted()) setAnalysis(res.analysis ?? '');
      })
      .catch((err: Error & { response?: { data?: { error?: string } } }) => {
        if (checkMounted()) {
          const msg =
            err?.response?.data?.error ?? err?.message ?? 'Failed to get AI insights';
          setError(msg);
        }
      })
      .finally(() => {
        if (checkMounted()) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [hasUrlContext, urlServiceId, urlQueryText, queryId, fingerprint, timeFrom, timeTo, loading, analysis]);

  const handleRunManual = () => {
    if (!manualServiceId.trim() || !manualQueryText.trim()) return;
    runOnceRef.current = false;
    setAnalysis(null);
    setError(null);
    setLoading(true);
    adreQanInsights({
      serviceId: manualServiceId.trim(),
      queryText: manualQueryText.trim(),
    })
      .then((res) => {
        if (mountedRef.current) setAnalysis(res.analysis ?? '');
      })
      .catch((err: Error & { response?: { data?: { error?: string } } }) => {
        if (mountedRef.current) {
          setError(
            err?.response?.data?.error ?? err?.message ?? 'Failed to get AI insights'
          );
        }
      })
      .finally(() => {
        if (mountedRef.current) setLoading(false);
      });
  };

  return (
    <Page title="QAN AI Insights">
      <Stack gap={2} sx={{ maxWidth: 900 }}>
        {hasContext && (
          <Typography variant="body2" color="text.secondary">
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
              <Typography variant="subtitle1" fontWeight={600} gutterBottom>
                Analysis
              </Typography>
              <Typography
                component="div"
                variant="body2"
                sx={{ '& p': { mb: 1 }, '& pre': { overflow: 'auto' } }}
              >
                <Markdown
                  remarkPlugins={[remarkGfm]}
                  components={{
                    code: ({ children }) => <CodeBlock>{children}</CodeBlock>,
                  }}
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
