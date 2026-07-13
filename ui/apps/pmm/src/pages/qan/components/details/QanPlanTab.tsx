import { Stack, Typography } from '@mui/material';
import { FC } from 'react';
import { SyntaxHighlighter } from 'components/syntax-highlighter';
import { useQanPlan } from 'hooks/api/useQan';
import { useQanPanelState } from '../../hooks/useQanPanelState';
import { QanDetailsError, QanDetailsLoading } from '../QanSectionShared';

export const QanPlanTab: FC = () => {
  const state = useQanPanelState();
  const { data, isLoading, isError } = useQanPlan(state.queryId ?? '', !!state.queryId);

  if (isLoading) return <QanDetailsLoading />;
  if (isError) return <QanDetailsError message="Failed to load query plan." />;

  if (!data?.plan) {
    return <Typography color="text.secondary">No plan available for this query.</Typography>;
  }

  return (
    <Stack spacing={1}>
      <SyntaxHighlighter language="json" content={data.plan} showCopyButton />
    </Stack>
  );
};
