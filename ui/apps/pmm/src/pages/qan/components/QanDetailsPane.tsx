import {
  Alert,
  Box,
  CircularProgress,
  Stack,
  Tab,
  Tabs,
  Typography,
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import IconButton from '@mui/material/IconButton';
import { FC, useMemo } from 'react';
import { useQanPanelActions, useQanPanelState } from '../hooks/useQanPanelState';
import { getVisibleQanTabs } from '../utils/qanTabVisibility';
import { useQanDatabaseType } from '../hooks/useQanDatabaseType';
import type { QanDetailsTab } from 'types/qan.types';
import { QanDetailsTabPanel } from './details/QanDetailsTabPanel';
import { QanExamplesTab } from './details/QanExamplesTab';
import { QanExplainTab } from './details/QanExplainTab';
import { QanTablesTab } from './details/QanTablesTab';
import { QanPlanTab } from './details/QanPlanTab';
import { QanAiInsightsTab } from './details/QanAiInsightsTab';

const TAB_LABELS: Record<QanDetailsTab, string> = {
  details: 'Details',
  examples: 'Examples',
  explain: 'Explain',
  tables: 'Tables',
  plan: 'Plan',
  aiInsights: 'AI Insights',
};

export const QanDetailsPane: FC = () => {
  const state = useQanPanelState();
  const actions = useQanPanelActions();
  const databaseType = useQanDatabaseType(state.labels, state.database);
  const visible = getVisibleQanTabs(state, databaseType);

  const tabs = useMemo(
    () =>
      (Object.keys(TAB_LABELS) as QanDetailsTab[]).filter((k) => visible[k]),
    [visible]
  );

  const activeTab = tabs.includes(state.openDetailsTab)
    ? state.openDetailsTab
    : tabs[0] ?? 'details';

  return (
    <Stack sx={{ flex: 1, minHeight: 0 }} data-testid="query-analytics-details">
      <Stack direction="row" alignItems="center" justifyContent="space-between">
        <Tabs
          value={activeTab}
          onChange={(_, v: QanDetailsTab) => actions.setTab(v)}
          variant="scrollable"
          scrollButtons="auto"
        >
          {tabs.map((t) => (
            <Tab key={t} value={t} label={TAB_LABELS[t]} />
          ))}
        </Tabs>
        <IconButton onClick={() => actions.closeDetails()} aria-label="Close details">
          <CloseIcon />
        </IconButton>
      </Stack>
      <Box sx={{ flex: 1, overflow: 'auto', p: 2 }}>
        {!state.queryId ? (
          <Alert severity="info">Select a query from the list above.</Alert>
        ) : (
          <>
            {activeTab === 'details' ? <QanDetailsTabPanel /> : null}
            {activeTab === 'examples' ? <QanExamplesTab /> : null}
            {activeTab === 'explain' ? <QanExplainTab /> : null}
            {activeTab === 'tables' ? <QanTablesTab /> : null}
            {activeTab === 'plan' ? <QanPlanTab /> : null}
            {activeTab === 'aiInsights' ? <QanAiInsightsTab /> : null}
          </>
        )}
      </Box>
    </Stack>
  );
};

export const QanDetailsLoading: FC = () => (
  <Box sx={{ display: 'flex', justifyContent: 'center', p: 3 }}>
    <CircularProgress size={28} />
  </Box>
);

export const QanDetailsError: FC<{ message?: string }> = ({ message }) => (
  <Typography color="error">{message ?? 'Failed to load data.'}</Typography>
);
