import { Navigate, useSearchParams } from 'react-router-dom';
import { useSettings } from 'contexts/settings';
import { PMM_NEW_NAV_PATH } from 'lib/constants';
import {
  Alert,
  Button,
  Card,
  CardContent,
  CircularProgress,
  Divider,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Stack,
  TextField,
  Typography,
} from '@mui/material';
import RefreshIcon from '@mui/icons-material/Refresh';
import { FC, useCallback, useEffect, useRef, useState } from 'react';
import Markdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import rehypeRaw from 'rehype-raw';
import { Page } from 'components/page';
import { adreQanInsights, getQanInsightsCache } from 'api/adre';
import { getMarkdownComponents } from 'components/adre/adre-chat-markdown.helpers';
import { holmesUsageSummaryLine } from 'utils/holmesUsageFormat';

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

type AnalysisSection = { title: string; content: string };

function parseAnalysisSections(raw: string): AnalysisSection[] {
  const text = raw.trim();
  if (!text) return [];

  const markdownHeadings = text.match(/^##\s+.+$/gm);
  if (markdownHeadings && markdownHeadings.length > 0) {
    const sections: AnalysisSection[] = [];
    const re = /^##\s+(.+)$/gm;
    const matches = [...text.matchAll(re)];
    for (let i = 0; i < matches.length; i += 1) {
      const start = matches[i].index ?? 0;
      const nextStart = matches[i + 1]?.index ?? text.length;
      const title = matches[i][1].trim();
      const content = text.slice(start + matches[i][0].length, nextStart).trim();
      sections.push({ title, content });
    }
    return sections;
  }

  const lines = text.split('\n');
  const marker = /^\s*(?:[-*•]\s*)?(Summary|Evidence|Recommendations?|EXPLAIN output|SHOW INDEX|SHOW CREATE TABLE)\s*:?\s*$/i;
  const sections: AnalysisSection[] = [];
  let currentTitle = 'Analysis';
  let buffer: string[] = [];

  for (const line of lines) {
    const m = line.match(marker);
    if (m) {
      if (buffer.join('\n').trim()) {
        sections.push({ title: currentTitle, content: buffer.join('\n').trim() });
      }
      currentTitle = m[1].trim();
      buffer = [];
    } else {
      buffer.push(line);
    }
  }
  if (buffer.join('\n').trim()) {
    sections.push({ title: currentTitle, content: buffer.join('\n').trim() });
  }

  return sections.length > 0 ? sections : [{ title: 'Analysis', content: text }];
}

function parsePipeTable(text: string): string[][] {
  const rows = text
    .split('\n')
    .map((line) => line.trim())
    .filter((line) => line.includes('|'))
    .map((line) => line.split('|').map((cell) => cell.trim()).filter((cell) => cell.length > 0))
    .filter((cells) => cells.length > 1);
  if (rows.length < 2) return [];

  return rows.filter((r) => !r.every((c) => /^-+$/.test(c)));
}

function isStructuredSection(title: string): boolean {
  const t = title.toLowerCase();
  return t.includes('explain') || t.includes('show index') || t.includes('show create table');
}

const QanAiInsightsPageContent: FC = () => {
  const [searchParams] = useSearchParams();
  const [analysis, setAnalysis] = useState<string | null>(null);
  const [cachedAt, setCachedAt] = useState<string | null>(null);
  const [usageLine, setUsageLine] = useState<string | null>(null);
  const [fromCache, setFromCache] = useState(false);
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
  const analysisSections = analysis ? parseAnalysisSections(analysis) : [];

  const runAnalysis = useCallback((force: boolean) => {
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
          setCachedAt(res.created_at ?? res.createdAt ?? null);
          setFromCache(!!res.cached);
          const u = res.usage;
          setUsageLine(u ? holmesUsageSummaryLine({
            model: u.model,
            totalTokens: u.totalTokens ?? u.total_tokens,
            cachedTokens: u.cachedTokens ?? u.cached_tokens,
            totalCost: u.totalCost ?? u.total_cost,
          }) : null);
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
  }, [serviceId, queryText, queryId, fingerprint, timeFrom, timeTo]);

  useEffect(() => {
    // runOnceRef enforces single auto-run per URL-context session; handleRunManual resets it.
    if (!hasUrlContext || runOnceRef.current) return;
    runOnceRef.current = true;

    if (queryId) {
      setLoading(true);
      getQanInsightsCache(queryId, urlServiceId.trim())
        .then((cached) => {
          if (!mountedRef.current) return;
          if (cached?.analysis) {
            setAnalysis(cached.analysis);
            setCachedAt(cached.created_at ?? cached.createdAt ?? null);
            setFromCache(true);
            const u = cached.usage;
            setUsageLine(u ? holmesUsageSummaryLine({
              model: u.model,
              totalTokens: u.totalTokens ?? u.total_tokens,
              cachedTokens: u.cachedTokens ?? u.cached_tokens,
              totalCost: u.totalCost ?? u.total_cost,
            }) : null);
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
  }, [hasUrlContext, urlServiceId, urlQueryText, queryId, fingerprint, timeFrom, timeTo, runAnalysis]);

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
          <Stack gap={1.25}>
            <Card variant="outlined">
              <CardContent>
                <Stack direction="row" alignItems="center" justifyContent="space-between" sx={{ mb: 1 }}>
                  <Stack>
                    <Typography variant="subtitle1" fontWeight={600}>
                      AI Insights
                    </Typography>
                    {cachedAt && (
                      <Typography variant="caption" color="text.secondary">
                        Last analyzed: {formatCacheTimestamp(cachedAt)}
                        {fromCache ? ' (cached)' : ''}
                      </Typography>
                    )}
                    {usageLine && (
                      <Typography variant="caption" color="text.secondary" display="block">
                        {usageLine}
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
                <Divider />
              </CardContent>
            </Card>

            {analysisSections.map((section, idx) => {
              const tableRows = parsePipeTable(section.content);
              const showTable = tableRows.length >= 2 && isStructuredSection(section.title);

              return (
                <Card variant="outlined" key={`${section.title}-${idx}`}>
                  <CardContent>
                    <Typography variant="subtitle2" fontWeight={600} sx={{ mb: 1 }}>
                      {section.title}
                    </Typography>
                    {showTable ? (
                      <TableContainer>
                        <Table size="small">
                          <TableHead>
                            <TableRow>
                              {tableRows[0].map((h, i) => (
                                <TableCell key={`${h}-${i}`} sx={{ fontWeight: 600 }}>
                                  {h}
                                </TableCell>
                              ))}
                            </TableRow>
                          </TableHead>
                          <TableBody>
                            {tableRows.slice(1).map((row, rowIdx) => (
                              <TableRow key={rowIdx}>
                                {row.map((cell, i) => (
                                  <TableCell key={`${rowIdx}-${i}`}>
                                    <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>
                                      {cell}
                                    </Typography>
                                  </TableCell>
                                ))}
                              </TableRow>
                            ))}
                          </TableBody>
                        </Table>
                      </TableContainer>
                    ) : (
                      <Typography
                        component="div"
                        variant="body2"
                        sx={{ '& p': { mb: 1 }, '& pre': { overflow: 'auto' } }}
                      >
                        <Markdown
                          remarkPlugins={[remarkGfm]}
                          rehypePlugins={[rehypeRaw]}
                          components={getMarkdownComponents(section.content)}
                        >
                          {section.content}
                        </Markdown>
                      </Typography>
                    )}
                  </CardContent>
                </Card>
              );
            })}
          </Stack>
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

const QanAiInsightsPage: FC = () => {
  const { settings } = useSettings();
  const [searchParams] = useSearchParams();
  if (settings?.nativeQanEnabled) {
    const next = new URLSearchParams(searchParams);
    next.set('tab', 'aiInsights');
    return <Navigate to={`${PMM_NEW_NAV_PATH}/qan?${next.toString()}`} replace />;
  }
  return <QanAiInsightsPageContent />;
};

export default QanAiInsightsPage;
