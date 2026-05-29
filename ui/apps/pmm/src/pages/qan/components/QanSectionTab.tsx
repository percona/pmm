import {
  Alert,
  Box,
  IconButton,
  Stack,
  Tab,
  Tabs,
  Typography,
} from '@mui/material';
import AutoAwesomeIcon from '@mui/icons-material/AutoAwesome';
import CloseIcon from '@mui/icons-material/Close';
import KeyboardArrowDownIcon from '@mui/icons-material/KeyboardArrowDown';
import KeyboardArrowUpIcon from '@mui/icons-material/KeyboardArrowUp';
import OpenInFullIcon from '@mui/icons-material/OpenInFull';
import { FC, useEffect, useMemo } from 'react';
import { useQanPanelActions, useQanPanelState } from '../hooks/useQanPanelState';
import { getVisibleQanTabs } from '../utils/qanTabVisibility';
import { useQanDatabaseType } from '../hooks/useQanDatabaseType';
import type { QanDetailsTab } from 'types/qan.types';
import { QAN_SECTION_TAB_LABELS, QAN_SECTION_TAB_ORDER } from '../utils/qanSectionTabs';
import { QanDetailsOverview } from './details/QanDetailsOverview';
import { QanExamplesTab } from './details/QanExamplesTab';
import { QanExplainPlanTab } from './details/QanExplainPlanTab';
import { QanTablesTab } from './details/QanTablesTab';
import { QanAiInsightsTab } from './details/QanAiInsightsTab';

const tabSx = {
  minHeight: 40,
  py: 1,
  px: 1.5,
  fontWeight: 600,
  fontSize: 15,
  textTransform: 'none' as const,
};

export const QanSectionTab: FC = () => {
  const state = useQanPanelState();
  const actions = useQanPanelActions();
  const databaseType = useQanDatabaseType(state.labels, state.database);
  const visible = getVisibleQanTabs(state, databaseType);

  const tabs = useMemo(
    () => QAN_SECTION_TAB_ORDER.filter((k) => visible[k]),
    [visible]
  );

  const activeTab = tabs.includes(state.openDetailsTab)
    ? state.openDetailsTab
    : tabs[0] ?? 'details';

  useEffect(() => {
    if (tabs.length && state.openDetailsTab !== activeTab) {
      actions.setTab(activeTab);
    }
  }, [tabs, state.openDetailsTab, activeTab, actions]);

  const showMoreListing = () =>
    actions.setSplitRatio(Math.min(actions.getSplitRatio() + 0.1, 0.75));
  const showMoreDetails = () =>
    actions.setSplitRatio(Math.max(actions.getSplitRatio() - 0.1, 0.25));
  const maximizeDetails = () => actions.setSplitRatio(0.25);

  return (
    <Stack
      sx={{
        flex: 1,
        minHeight: 0,
        bgcolor: 'background.default',
        borderTopLeftRadius: 8,
        borderTopRightRadius: 8,
        borderTop: 1,
        borderColor: 'divider',
        pt: 1,
        px: 2,
      }}
      data-testid="query-analytics-details"
    >
      <Stack
        direction="row"
        alignItems="flex-end"
        justifyContent="space-between"
        sx={{
          borderBottom: 1,
          borderColor: 'divider',
          minHeight: 40,
          gap: 3,
        }}
      >
        <Typography
          variant="h5"
          sx={{ fontWeight: 600, flexShrink: 0, pb: 0.75, lineHeight: 1.125 }}
        >
          Query Fingerprint
        </Typography>
        <Tabs
          value={activeTab}
          onChange={(_, v: QanDetailsTab) => actions.setTab(v)}
          variant="scrollable"
          scrollButtons="auto"
          sx={{
            flex: 1,
            minWidth: 0,
            minHeight: 40,
            '& .MuiTabs-indicator': { height: 3 },
          }}
        >
          {tabs.map((t) => (
            <Tab
              key={t}
              value={t}
              sx={tabSx}
              label={
                t === 'aiInsights' ? (
                  <Stack direction="row" alignItems="center" spacing={0.75} component="span">
                    <AutoAwesomeIcon sx={{ fontSize: 20 }} />
                    <span>{QAN_SECTION_TAB_LABELS[t]}</span>
                  </Stack>
                ) : (
                  QAN_SECTION_TAB_LABELS[t]
                )
              }
            />
          ))}
        </Tabs>
        <Stack direction="row" alignItems="center" sx={{ flexShrink: 0, pb: 0.25 }}>
          <IconButton size="small" onClick={showMoreListing} aria-label="Show more listing">
            <KeyboardArrowUpIcon fontSize="small" />
          </IconButton>
          <IconButton size="small" onClick={showMoreDetails} aria-label="Show more details">
            <KeyboardArrowDownIcon fontSize="small" />
          </IconButton>
          <IconButton size="small" onClick={maximizeDetails} aria-label="Maximize details">
            <OpenInFullIcon fontSize="small" />
          </IconButton>
          <IconButton onClick={() => actions.closeDetails()} aria-label="Close details" size="small">
            <CloseIcon fontSize="small" />
          </IconButton>
        </Stack>
      </Stack>
      <Box sx={{ flex: 1, overflow: 'auto', py: 1 }}>
        {!state.queryId ? (
          <Alert severity="info">Select a query from the list above.</Alert>
        ) : (
          <>
            {activeTab === 'details' ? <QanDetailsOverview /> : null}
            {activeTab === 'examples' ? <QanExamplesTab /> : null}
            {activeTab === 'explainPlan' ? <QanExplainPlanTab /> : null}
            {activeTab === 'tables' ? <QanTablesTab /> : null}
            {activeTab === 'aiInsights' ? <QanAiInsightsTab /> : null}
          </>
        )}
      </Box>
    </Stack>
  );
};
