import { Alert, Stack, Typography } from '@mui/material';
import { FC, useMemo, useState } from 'react';
import { SyntaxHighlighter } from 'components/syntax-highlighter';
import { useQanExamples, useQanExplain } from 'hooks/api/useQan';
import { useQanPanelState } from '../../hooks/useQanPanelState';
import { getLabelQueryParams } from '../../utils/qanTools';
import { useQanServiceId } from '../../hooks/useQanServiceId';
import { QanDetailsError, QanDetailsLoading } from '../QanSectionShared';

export const QanExplainTab: FC = () => {
  const state = useQanPanelState();
  const [exampleIdx] = useState(0);
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
  const serviceId = useQanServiceId(exData?.examples);
  const example = exData?.examples?.[exampleIdx];

  const explainParams = useMemo(
    () => ({
      queryId: state.queryId ?? '',
      serviceId,
      database: state.database,
      example: example?.example,
    }),
    [state, serviceId, example]
  );

  const { data, isLoading, isError } = useQanExplain(
    explainParams,
    !!state.queryId && !!serviceId
  );

  if (exLoading || isLoading) return <QanDetailsLoading />;
  if (isError) return <QanDetailsError message="Failed to load EXPLAIN output." />;
  if (!serviceId) {
    return (
      <Alert severity="info">
        Select a service filter or ensure examples include a service id to run EXPLAIN.
      </Alert>
    );
  }

  return (
    <Stack spacing={2}>
      <Typography variant="caption" color="text.secondary">
        Advisory: review EXPLAIN output before applying any suggested changes in your environment.
      </Typography>
      {data?.classic ? (
        <Stack>
          <Typography variant="subtitle2">Classic</Typography>
          <SyntaxHighlighter language="text" content={data.classic} showCopyButton />
        </Stack>
      ) : null}
      {data?.json ? (
        <Stack>
          <Typography variant="subtitle2">JSON</Typography>
          <SyntaxHighlighter language="json" content={data.json} showCopyButton />
        </Stack>
      ) : null}
      {data?.visual ? (
        <Stack>
          <Typography variant="subtitle2">Visual</Typography>
          <SyntaxHighlighter language="json" content={data.visual} showCopyButton />
        </Stack>
      ) : null}
      {!data?.classic && !data?.json && !data?.visual ? (
        <Typography color="text.secondary">No EXPLAIN data available.</Typography>
      ) : null}
    </Stack>
  );
};
