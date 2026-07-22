import { useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Skeleton,
  Stack,
  tableContainerClasses,
  Typography,
} from '@mui/material';
import { usePrometheusAlertRules } from 'hooks/api/usePrometheusAlertRules';
import { useAlertmanagerAlerts } from 'hooks/api/useAlertmanagerAlerts';
import { useAlertmanagerSilences } from 'hooks/api/useAlertmanagerSilences';
import { buildSilenceMap, flattenAlertRules } from './AlertsPage.utils';
import { AlertStatusTable } from './table';
import { useDetailsPaneNavigation } from '@percona/percona-ui';
import { AlertRow, AlertsTableRow } from './AlertsPage.types';
import { AlertDetailsPane } from './details-pane';
import { Messages } from './AlertsPage.messages';
import { updateDocumentTitle } from 'utils/document.utils';
import { useSettings } from 'contexts/settings';
import FeatureCheck from 'components/feature-check';

const AlertsPage = () => {
  updateDocumentTitle(Messages.browserTitle);
  const { settings } = useSettings();
  const { data, isError, isLoading } = usePrometheusAlertRules({
    refetchInterval: 5000,
    enabled: settings?.alertingEnabled,
  });
  // Supplementary: silence data enriches the rows but must not block rendering.
  // If this call fails the page still lists alerts, just without silenced badges.
  const { data: silencedAlerts } = useAlertmanagerAlerts({
    refetchInterval: 5000,
    enabled: settings?.alertingEnabled,
  });
  const { data: silences } = useAlertmanagerSilences({
    refetchInterval: 5000,
    enabled: settings?.alertingEnabled,
  });
  const silenceMap = useMemo(
    () => buildSilenceMap(silencedAlerts, silences),
    [silencedAlerts, silences]
  );
  const rows = useMemo(
    () => flattenAlertRules(data, silenceMap),
    [data, silenceMap]
  );
  const [navigableRows, setNavigableRows] = useState<AlertsTableRow[]>(rows);
  const [selectedRowId, setSelectedRowId] = useState<string>();
  // Node group headers are not navigable — the details pane only shows alerts, so
  // next/previous must step over them (and collapsed groups' hidden alerts drop out).
  const navigableAlerts = useMemo(
    () => navigableRows.filter((row): row is AlertRow => row.type === 'alert'),
    [navigableRows]
  );
  const selectedRow = useMemo(
    () =>
      navigableAlerts.some((row) => row.id === selectedRowId)
        ? rows.find((row) => row.id === selectedRowId)
        : undefined,
    [rows, navigableAlerts, selectedRowId]
  );

  useEffect(() => {
    if (selectedRowId && !selectedRow) {
      setSelectedRowId(undefined);
    }
  }, [selectedRow, selectedRowId]);

  const detailsPaneProps = useDetailsPaneNavigation<AlertRow>({
    rows: navigableAlerts,
    selected: selectedRow,
    getRowId: (row) => row.id,
    onSelect: (row) => setSelectedRowId(row?.id),
  });

  if (!settings?.alertingEnabled) {
    return (
      <FeatureCheck pageTitle={Messages.title} feature={Messages.feature} />
    );
  }

  return (
    <Stack
      direction="column"
      sx={{
        flex: 1,
        gap: 2,
        m: 2,
        mb: 0,
      }}
    >
      <Typography variant="h3">{Messages.title}</Typography>
      <Stack flex="1" position="relative">
        {isLoading ? (
          <Skeleton
            aria-label={Messages.loading}
            variant="rounded"
            height="70vh"
          />
        ) : isError && !data ? (
          <Alert severity="error">{Messages.fetchError}</Alert>
        ) : (
          <Stack
            sx={(theme) => ({
              flex: 1,
              maxHeight: '92vh',

              '& > *': {
                flex: 1,
                display: 'flex',
                flexDirection: 'column',
              },

              [`.${tableContainerClasses.root}`]: {
                flex: 1,
                borderWidth: 1,
                borderStyle: 'solid',
                borderColor: theme.palette.divider,
                borderRadius: 1,
              },
            })}
          >
            <AlertStatusTable
              rows={rows}
              onNavigableRowsChange={setNavigableRows}
              onOpenDetail={(row) => setSelectedRowId(row.id)}
            />
          </Stack>
        )}
        <AlertDetailsPane
          alert={selectedRow}
          onClose={() => setSelectedRowId(undefined)}
          {...detailsPaneProps}
        />
      </Stack>
    </Stack>
  );
};

export default AlertsPage;
