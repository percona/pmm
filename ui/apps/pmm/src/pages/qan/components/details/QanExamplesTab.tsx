import { Stack, Typography } from '@mui/material';
import { FC, useMemo } from 'react';
import { SyntaxHighlighter } from 'components/syntax-highlighter';
import { useQanExamples } from 'hooks/api/useQan';
import { useQanPanelState } from '../../hooks/useQanPanelState';
import { getLabelQueryParams } from '../../utils/qanTools';
import { QanDetailsError, QanDetailsLoading } from '../QanSectionShared';

export const QanExamplesTab: FC = () => {
  const state = useQanPanelState();
  const params = useMemo(
    () => ({
      filterBy: state.queryId ?? '',
      groupBy: state.groupBy,
      labels: getLabelQueryParams(state.labels),
      periodStartFrom: state.from,
      periodStartTo: state.to,
    }),
    [state]
  );
  const { data, isLoading, isError } = useQanExamples(params, !!state.queryId);

  if (isLoading) return <QanDetailsLoading />;
  if (isError) return <QanDetailsError />;

  const examples = Array.isArray(data?.examples) ? data.examples : [];
  if (!examples.length) {
    return <Typography color="text.secondary">No examples found.</Typography>;
  }

  return (
    <Stack spacing={2}>
      {examples.map((ex, i) => (
        <Stack key={i} spacing={1}>
          {ex.exampleType ? (
            <Typography variant="caption" color="text.secondary">
              {ex.exampleType}
            </Typography>
          ) : null}
          <SyntaxHighlighter language="text" content={ex.example ?? ''} showCopyButton />
        </Stack>
      ))}
    </Stack>
  );
};
