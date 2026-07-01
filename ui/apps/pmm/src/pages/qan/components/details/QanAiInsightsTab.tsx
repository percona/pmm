import {
  Alert,
  Button,
  CircularProgress,
  Divider,
  Stack,
  Typography,
} from '@mui/material';
import RefreshIcon from '@mui/icons-material/Refresh';
import { FC, useCallback, useEffect, useMemo, useState } from 'react';
import Markdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import rehypeRaw from 'rehype-raw';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { adreQanInsights, getQanInsightsCache } from 'api/adre';
import { getMarkdownComponents } from 'components/adre/adre-chat-markdown.helpers';
import { holmesUsageSummaryLine } from 'utils/holmesUsageFormat';
import { useQanExamples } from 'hooks/api/useQan';
import { useQanPanelState } from '../../hooks/useQanPanelState';
import { useQanServiceId } from '../../hooks/useQanServiceId';
import { getLabelQueryParams } from '../../utils/qanTools';

const RUNNING_MESSAGE =
  'Query analysis is running. Results will appear here soon. Recommendations are advisory — copy and apply manually in your environment.';

export const QanAiInsightsTab: FC = () => {
  const state = useQanPanelState();
  const queryClient = useQueryClient();
  const [refreshing, setRefreshing] = useState(false);

  const exampleParams = useMemo(
    () => ({
      filterBy: state.queryId ?? '',
      groupBy: state.groupBy,
      labels: getLabelQueryParams(state.labels),
      periodStartFrom: state.from,
      periodStartTo: state.to,
    }),
    [state]
  );
  const { data: exData } = useQanExamples(exampleParams, !!state.queryId && !state.totals);
  const serviceId = useQanServiceId(exData?.examples);

  const cacheKey = useMemo(
    () => ['qan', 'ai-insights', serviceId, state.queryId, state.from, state.to] as const,
    [serviceId, state.queryId, state.from, state.to]
  );

  const { data, isLoading, isFetching, error, refetch } = useQuery({
    queryKey: cacheKey,
    queryFn: async () => {
      if (!serviceId || !state.queryId) return null;
      const cached = await getQanInsightsCache(state.queryId, serviceId);
      if (cached?.analysis) return cached;
      return adreQanInsights({
        serviceId,
        queryId: state.queryId,
        queryText: state.fingerprint ?? '',
        fingerprint: state.fingerprint,
        timeFrom: state.from,
        timeTo: state.to,
      });
    },
    enabled: !!serviceId && !!state.queryId && !state.totals,
  });

  const onRefresh = useCallback(async () => {
    if (!serviceId || !state.queryId) return;
    setRefreshing(true);
    try {
      await adreQanInsights({
        serviceId,
        queryId: state.queryId,
        queryText: state.fingerprint ?? '',
        fingerprint: state.fingerprint,
        timeFrom: state.from,
        timeTo: state.to,
        force: true,
      });
      await queryClient.invalidateQueries({ queryKey: cacheKey });
      await refetch();
    } finally {
      setRefreshing(false);
    }
  }, [serviceId, state, queryClient, cacheKey, refetch]);

  const analysis = data?.analysis ?? '';
  const pending = !analysis && (isLoading || isFetching);

  useEffect(() => {
    if (!pending) return undefined;
    const t = window.setInterval(() => {
      void refetch();
    }, 5000);
    return () => clearInterval(t);
  }, [pending, refetch]);

  if (state.totals) {
    return (
      <Alert severity="info">AI Insights are not available for the totals row.</Alert>
    );
  }

  if (!serviceId) {
    return (
      <Alert severity="info">
        Select a service filter or open a query with examples to run AI Insights.
      </Alert>
    );
  }

  if (isLoading && !data) {
    return (
      <Stack alignItems="center" py={4}>
        <CircularProgress size={28} />
      </Stack>
    );
  }

  if (error) {
    return <Alert severity="error">Failed to load AI insights.</Alert>;
  }

  const usageLine = data?.usage ? holmesUsageSummaryLine(data.usage) : '';

  return (
    <Stack spacing={2}>
      <Alert severity="info" variant="outlined">
        AI recommendations are advisory only. Copy suggested SQL and review before executing in
        your database.
      </Alert>
      <Stack direction="row" justifyContent="flex-end">
        <Button
          size="small"
          startIcon={<RefreshIcon />}
          onClick={onRefresh}
          disabled={refreshing}
        >
          Refresh analysis
        </Button>
      </Stack>
      {pending ? (
        <Typography color="text.secondary">{RUNNING_MESSAGE}</Typography>
      ) : (
        <Markdown
          remarkPlugins={[remarkGfm]}
          rehypePlugins={[rehypeRaw]}
          components={getMarkdownComponents(analysis)}
        >
          {analysis || '_No analysis yet._'}
        </Markdown>
      )}
      {usageLine ? (
        <>
          <Divider />
          <Typography variant="caption" color="text.secondary">
            {usageLine}
          </Typography>
        </>
      ) : null}
    </Stack>
  );
};
