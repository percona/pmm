import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  CircularProgress,
  FormControl,
  InputLabel,
  MenuItem,
  Select,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
} from '@mui/material';
import { FC, useEffect, useMemo, useRef, useState } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import { Page } from 'components/page';
import { useAdreUsageEvents, useAdreUsageSummary } from 'hooks/api/useAdreUsage';
import { PMM_NEW_NAV_PATH } from 'lib/constants';
import {
  aggregateUsageSeriesByDay,
  fillDailyCostSeries,
  formatTokenCount,
  formatTokensWithCached,
  formatUsdCost,
  formatUsageDayLabel,
  HOLMES_FEATURE_LABELS,
} from 'utils/holmesUsageFormat';

type RangePreset = '7d' | '30d' | '90d';

function rangeFromPreset(preset: RangePreset): { from: string; to: string } {
  const to = new Date();
  const from = new Date();
  const days = preset === '7d' ? 7 : preset === '90d' ? 90 : 30;
  from.setDate(from.getDate() - days);
  return { from: from.toISOString(), to: to.toISOString() };
}

function num(v: number | undefined, fallback?: number): number {
  return v ?? fallback ?? 0;
}

const AdreUsagePage: FC = () => {
  const [preset, setPreset] = useState<RangePreset>('30d');
  const [featureFilter, setFeatureFilter] = useState('');
  const range = useMemo(() => rangeFromPreset(preset), [preset]);
  const summaryQuery = useAdreUsageSummary({
    ...range,
    groupBy: 'day',
    feature: featureFilter || undefined,
  });
  const eventsQuery = useAdreUsageEvents({
    ...range,
    limit: 100,
    feature: featureFilter || undefined,
  });

  const totals = summaryQuery.data?.totals;
  const series = summaryQuery.data?.series ?? [];
  const byFeature = summaryQuery.data?.byFeature ?? summaryQuery.data?.by_feature ?? [];
  const byModel = summaryQuery.data?.byModel ?? summaryQuery.data?.by_model ?? [];
  const events = eventsQuery.data?.events ?? [];

  const dailyCostSeries = useMemo(() => {
    const from = summaryQuery.data?.from ?? range.from;
    const to = summaryQuery.data?.to ?? range.to;
    return fillDailyCostSeries(aggregateUsageSeriesByDay(series), from, to);
  }, [series, summaryQuery.data?.from, summaryQuery.data?.to, range.from, range.to]);

  const maxSeriesCost = Math.max(...dailyCostSeries.map((s) => s.totalCost), 0.0001);
  const hasAnyDailyCost = dailyCostSeries.some((s) => s.totalCost > 0);
  const costSeriesRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    costSeriesRef.current?.scrollTo({ top: 0 });
  }, [preset, featureFilter, dailyCostSeries.length, dailyCostSeries[0]?.bucket]);

  const exportCsv = () => {
    const url = `/v1/adre/usage/events?from=${encodeURIComponent(range.from)}&to=${encodeURIComponent(range.to)}&format=csv&limit=500${featureFilter ? `&feature=${encodeURIComponent(featureFilter)}` : ''}`;
    window.open(url, '_blank', 'noopener,noreferrer');
  };

  return (
    <Page title="AI Usage">
      <Stack spacing={2} sx={{ mb: 2, overflow: 'visible' }}>
        <Typography variant="body2" color="text.secondary">
          Token and cost usage across PMM AI features
        </Typography>
        <Stack direction="row" spacing={1.5} alignItems="center" flexWrap="wrap" useFlexGap>
          <FormControl size="small" sx={{ minWidth: 140 }}>
            <InputLabel id="usage-range-label">Range</InputLabel>
            <Select
              labelId="usage-range-label"
              label="Range"
              value={preset}
              onChange={(e) => setPreset(e.target.value as RangePreset)}
            >
              <MenuItem value="7d">Last 7 days</MenuItem>
              <MenuItem value="30d">Last 30 days</MenuItem>
              <MenuItem value="90d">Last 90 days</MenuItem>
            </Select>
          </FormControl>
          <FormControl size="small" sx={{ minWidth: 160 }}>
            <InputLabel id="usage-feature-label">Feature</InputLabel>
            <Select
              labelId="usage-feature-label"
              label="Feature"
              value={featureFilter}
              onChange={(e) => setFeatureFilter(e.target.value)}
              displayEmpty
              renderValue={(value) =>
                value ? (HOLMES_FEATURE_LABELS[value] ?? value) : 'All'
              }
            >
              <MenuItem value="">All</MenuItem>
              {Object.entries(HOLMES_FEATURE_LABELS).map(([k, label]) => (
                <MenuItem key={k} value={k}>
                  {label}
                </MenuItem>
              ))}
            </Select>
          </FormControl>
          <Button size="small" variant="outlined" onClick={exportCsv} sx={{ flexShrink: 0 }}>
            Export CSV
          </Button>
        </Stack>
      </Stack>

      {summaryQuery.isLoading ? (
        <Box sx={{ display: 'flex', justifyContent: 'center', py: 6 }}>
          <CircularProgress />
        </Box>
      ) : summaryQuery.isError ? (
        <Alert severity="error">Failed to load usage summary.</Alert>
      ) : (
        <>
          <Stack direction={{ xs: 'column', md: 'row' }} spacing={2} sx={{ mb: 3 }}>
            {[
              { label: 'Total cost', value: formatUsdCost(num(totals?.totalCost ?? totals?.total_cost)) || '$0' },
              {
                label: 'Total tokens',
                value: formatTokenCount(num(totals?.totalTokens ?? totals?.total_tokens)) || '0',
              },
              {
                label: 'Cached tokens',
                value: `${formatTokenCount(num(totals?.cachedTokens ?? totals?.cached_tokens)) || '0'}${
                  totals && num(totals.totalTokens ?? totals.total_tokens) > 0
                    ? ` (${Math.round((100 * num(totals.cachedTokens ?? totals.cached_tokens)) / num(totals.totalTokens ?? totals.total_tokens))}%)`
                    : ''
                }`,
              },
              { label: 'AI calls', value: String(num(totals?.callCount ?? totals?.call_count)) },
            ].map((card) => (
              <Card key={card.label} variant="outlined" sx={{ flex: 1 }}>
                <CardContent>
                  <Typography variant="caption" color="text.secondary">
                    {card.label}
                  </Typography>
                  <Typography variant="h5">{card.value}</Typography>
                </CardContent>
              </Card>
            ))}
          </Stack>

          <Card variant="outlined" sx={{ mb: 3 }}>
            <CardContent>
              <Typography variant="subtitle2" sx={{ mb: 0.5 }}>
                Cost over time
              </Typography>
              <Typography variant="caption" color="text.secondary" display="block" sx={{ mb: 2 }}>
                Daily cost in the selected range (UTC days)
              </Typography>
              {!hasAnyDailyCost ? (
                <Typography variant="body2" color="text.secondary">
                  No usage recorded in this range.{' '}
                  <RouterLink to={`${PMM_NEW_NAV_PATH}/adre`}>Open AI Assistant</RouterLink>
                </Typography>
              ) : (
                <Box ref={costSeriesRef} sx={{ maxHeight: 280, overflow: 'auto', pr: 0.5 }}>
                  <Stack spacing={0.75}>
                    {dailyCostSeries.map((row) => {
                      const cost = row.totalCost;
                      const pct = cost > 0 ? Math.max(2, (cost / maxSeriesCost) * 100) : 0;
                      return (
                        <Stack key={row.bucket} direction="row" alignItems="center" spacing={1}>
                          <Typography
                            variant="caption"
                            sx={{ width: 56, flexShrink: 0, color: cost > 0 ? 'text.primary' : 'text.secondary' }}
                          >
                            {formatUsageDayLabel(row.bucket)}
                          </Typography>
                          <Box
                            sx={{
                              flex: 1,
                              minWidth: 0,
                              height: 10,
                              bgcolor: 'action.hover',
                              borderRadius: 1,
                            }}
                          >
                            {cost > 0 ? (
                              <Box
                                sx={{
                                  height: '100%',
                                  width: `${pct}%`,
                                  bgcolor: 'primary.main',
                                  borderRadius: 1,
                                }}
                              />
                            ) : null}
                          </Box>
                          <Typography
                            variant="caption"
                            color="text.secondary"
                            sx={{ width: 52, flexShrink: 0, textAlign: 'right' }}
                          >
                            {cost > 0 ? formatUsdCost(cost) : '—'}
                          </Typography>
                        </Stack>
                      );
                    })}
                  </Stack>
                </Box>
              )}
            </CardContent>
          </Card>

          <Stack direction={{ xs: 'column', md: 'row' }} spacing={2} sx={{ mb: 3 }}>
            <Card variant="outlined" sx={{ flex: 1 }}>
              <CardContent>
                <Typography variant="subtitle2" sx={{ mb: 1 }}>
                  By feature
                </Typography>
                {byFeature.map((row) => (
                  <Stack key={row.feature} direction="row" justifyContent="space-between" sx={{ py: 0.5 }}>
                    <Typography variant="body2">
                      {HOLMES_FEATURE_LABELS[row.feature ?? ''] ?? row.feature}
                    </Typography>
                    <Typography variant="body2" color="text.secondary">
                      {formatUsdCost(num(row.totalCost ?? row.total_cost))}
                    </Typography>
                  </Stack>
                ))}
              </CardContent>
            </Card>
            <Card variant="outlined" sx={{ flex: 1 }}>
              <CardContent>
                <Typography variant="subtitle2" sx={{ mb: 1 }}>
                  By model
                </Typography>
                {byModel.map((row) => (
                  <Stack key={row.model} direction="row" justifyContent="space-between" sx={{ py: 0.5 }}>
                    <Typography variant="body2">{row.model || 'default'}</Typography>
                    <Typography variant="body2" color="text.secondary">
                      {formatUsdCost(num(row.totalCost ?? row.total_cost))}
                    </Typography>
                  </Stack>
                ))}
              </CardContent>
            </Card>
          </Stack>

          <Typography variant="h6" sx={{ mb: 1 }}>
            Recent events
          </Typography>
          <TableContainer
            component={Card}
            variant="outlined"
            sx={{ maxHeight: 400, overflow: 'auto' }}
          >
            <Table size="small" stickyHeader>
              <TableHead>
                <TableRow>
                  <TableCell sx={{ bgcolor: 'background.paper' }}>Time</TableCell>
                  <TableCell sx={{ bgcolor: 'background.paper' }}>Feature</TableCell>
                  <TableCell sx={{ bgcolor: 'background.paper' }}>Model</TableCell>
                  <TableCell align="right" sx={{ bgcolor: 'background.paper' }}>
                    Tokens
                  </TableCell>
                  <TableCell align="right" sx={{ bgcolor: 'background.paper' }}>
                    Cost
                  </TableCell>
                  <TableCell sx={{ bgcolor: 'background.paper' }}>User</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {events.map((ev) => (
                  <TableRow key={ev.id}>
                    <TableCell>
                      {new Date(ev.createdAt ?? ev.created_at ?? '').toLocaleString()}
                    </TableCell>
                    <TableCell>{HOLMES_FEATURE_LABELS[ev.feature] ?? ev.feature}</TableCell>
                    <TableCell>{ev.model || 'default'}</TableCell>
                    <TableCell align="right">
                      {formatTokensWithCached(
                        ev.totalTokens ?? ev.total_tokens,
                        ev.cachedTokens ?? ev.cached_tokens
                      )}
                    </TableCell>
                    <TableCell align="right">
                      {formatUsdCost(ev.totalCost ?? ev.total_cost)}
                    </TableCell>
                    <TableCell>{ev.triggeredBy ?? ev.triggered_by ?? '—'}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </>
      )}
    </Page>
  );
};

export default AdreUsagePage;
