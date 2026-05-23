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
import { FC, useMemo, useState } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import { Page } from 'components/page';
import { useAdreUsageEvents, useAdreUsageSummary } from 'hooks/api/useAdreUsage';
import { PMM_NEW_NAV_PATH } from 'lib/constants';
import {
  formatTokenCount,
  formatTokensWithCached,
  formatUsdCost,
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

  const maxSeriesCost = Math.max(...series.map((s) => num(s.totalCost ?? s.total_cost)), 0.0001);

  const exportCsv = () => {
    const url = `/v1/adre/usage/events?from=${encodeURIComponent(range.from)}&to=${encodeURIComponent(range.to)}&format=csv&limit=500${featureFilter ? `&feature=${encodeURIComponent(featureFilter)}` : ''}`;
    window.open(url, '_blank', 'noopener,noreferrer');
  };

  return (
    <Page title="AI Usage">
      <Stack
        direction="row"
        alignItems="flex-end"
        justifyContent="space-between"
        flexWrap="wrap"
        gap={1}
        sx={{ mb: 2 }}
      >
        <Typography variant="body2" color="text.secondary">
          Token and cost usage across PMM AI features
        </Typography>
        <Stack direction="row" spacing={1} alignItems="flex-end" flexWrap="wrap">
          <FormControl size="small" sx={{ minWidth: 140 }}>
            <InputLabel id="usage-range-label" shrink>
              Range
            </InputLabel>
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
            <InputLabel id="usage-feature-label" shrink>
              Feature
            </InputLabel>
            <Select
              labelId="usage-feature-label"
              label="Feature"
              value={featureFilter}
              onChange={(e) => setFeatureFilter(e.target.value)}
            >
              <MenuItem value="">All</MenuItem>
              {Object.entries(HOLMES_FEATURE_LABELS).map(([k, label]) => (
                <MenuItem key={k} value={k}>
                  {label}
                </MenuItem>
              ))}
            </Select>
          </FormControl>
          <Button size="small" variant="outlined" onClick={exportCsv}>
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
              <Typography variant="subtitle2" sx={{ mb: 2 }}>
                Cost over time
              </Typography>
              {series.length === 0 ? (
                <Typography variant="body2" color="text.secondary">
                  No usage recorded in this range.{' '}
                  <RouterLink to={`${PMM_NEW_NAV_PATH}/adre`}>Open AI Assistant</RouterLink>
                </Typography>
              ) : (
                <Stack spacing={0.75}>
                  {series.map((row) => {
                    const cost = num(row.totalCost ?? row.total_cost);
                    const pct = Math.max(4, (cost / maxSeriesCost) * 100);
                    return (
                      <Stack key={row.bucket ?? row.feature} direction="row" alignItems="center" spacing={1}>
                        <Typography variant="caption" sx={{ width: 88, flexShrink: 0 }}>
                          {row.bucket}
                        </Typography>
                        <Box
                          sx={{
                            height: 10,
                            width: `${pct}%`,
                            minWidth: 4,
                            bgcolor: 'primary.main',
                            borderRadius: 1,
                          }}
                        />
                        <Typography variant="caption" color="text.secondary">
                          {formatUsdCost(cost)}
                        </Typography>
                      </Stack>
                    );
                  })}
                </Stack>
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
          <TableContainer component={Card} variant="outlined">
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell>Time</TableCell>
                  <TableCell>Feature</TableCell>
                  <TableCell>Model</TableCell>
                  <TableCell align="right">Tokens</TableCell>
                  <TableCell align="right">Cost</TableCell>
                  <TableCell>User</TableCell>
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
