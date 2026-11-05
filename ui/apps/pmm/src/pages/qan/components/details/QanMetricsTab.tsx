import { Stack, Typography } from '@mui/material';
import { FC, useMemo } from 'react';
import { useQanMetrics } from 'hooks/api/useQan';
import { useQanPanelState } from '../../hooks/useQanPanelState';
import { getLabelQueryParams } from '../../utils/qanTools';
import { QanDetailsError, QanDetailsLoading } from '../QanSectionShared';

export const QanMetricsTab: FC = () => {
  const state = useQanPanelState();
  const params = useMemo(
    () => ({
      filterBy: state.queryId ?? '',
      groupBy: state.groupBy,
      labels: getLabelQueryParams(state.labels),
      periodStartFrom: state.from,
      periodStartTo: state.to,
      totals: state.totals,
    }),
    [state]
  );
  const { data, isLoading, isError } = useQanMetrics(params, !!state.queryId);

  if (isLoading) return <QanDetailsLoading />;
  if (isError) return <QanDetailsError />;

  const textMetrics = data?.textMetrics ?? {};
  const metrics = data?.metrics ?? {};

  return (
    <Stack spacing={2}>
      {Object.entries(textMetrics).map(([k, v]) => (
        <Stack key={k}>
          <Typography variant="caption" color="text.secondary">
            {k}
          </Typography>
          <Typography variant="body2">{v}</Typography>
        </Stack>
      ))}
      {Object.entries(metrics).map(([name, mv]) => (
        <Stack key={name}>
          <Typography variant="subtitle2">{name}</Typography>
          <Typography variant="body2" color="text.secondary">
            avg: {mv.stats?.avg ?? '—'} · sum: {mv.stats?.sum ?? '—'} · rate:{' '}
            {mv.stats?.rate ?? '—'}
          </Typography>
        </Stack>
      ))}
      {data?.metadata ? (
        <Stack>
          <Typography variant="subtitle2">Metadata</Typography>
          <Typography
            component="pre"
            variant="body2"
            sx={{ whiteSpace: 'pre-wrap', fontFamily: 'monospace' }}
          >
            {JSON.stringify(data.metadata, null, 2)}
          </Typography>
        </Stack>
      ) : null}
    </Stack>
  );
};
