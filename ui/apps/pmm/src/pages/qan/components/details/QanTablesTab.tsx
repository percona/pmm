import { Stack, Typography } from '@mui/material';
import { FC, useMemo } from 'react';
import { useQanExamples, useQanSchema } from 'hooks/api/useQan';
import { useQanPanelState } from '../../hooks/useQanPanelState';
import { getLabelQueryParams } from '../../utils/qanTools';
import { QanDetailsError, QanDetailsLoading } from '../QanSectionShared';

export const QanTablesTab: FC = () => {
  const state = useQanPanelState();
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
  const { data: exData, isLoading: exLoading } = useQanExamples(exampleParams, !!state.queryId);
  const tables = useMemo(() => exData?.examples?.[0]?.tables ?? [], [exData]);

  const schemaParams = useMemo(
    () => ({
      filterBy: state.queryId ?? '',
      groupBy: state.groupBy,
      labels: getLabelQueryParams(state.labels),
      periodStartFrom: state.from,
      periodStartTo: state.to,
      tables,
    }),
    [state, tables]
  );

  const { data, isLoading, isError } = useQanSchema(
    schemaParams,
    !!state.queryId && tables.length > 0
  );

  if (exLoading || isLoading) return <QanDetailsLoading />;
  if (isError) return <QanDetailsError message="Failed to load table schema." />;

  if (!tables.length) {
    return <Typography color="text.secondary">No tables associated with this query.</Typography>;
  }

  return (
    <Stack spacing={2}>
      <Typography
        component="pre"
        variant="body2"
        sx={{ whiteSpace: 'pre-wrap', fontFamily: 'monospace' }}
      >
        {JSON.stringify(data?.tables ?? {}, null, 2)}
      </Typography>
    </Stack>
  );
};
