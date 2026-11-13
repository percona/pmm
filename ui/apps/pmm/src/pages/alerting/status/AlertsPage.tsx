import { useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
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
import {
  buildSilenceMap,
  findLinkedAlert,
  flattenAlertRules,
  withAlertLink,
} from './AlertsPage.utils';
import { ALERT_LINK_PARAMS } from './AlertsPage.constants';
import type { AlertDetailsPaneTab } from './details-pane/AlertDetailsPane.types';
import { AlertStatusTable } from './table';
import { useDetailsPaneNavigation } from '@percona/peak-ui';
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
  // Undefined until the table reports its rows, so a linked alert counts as
  // visible on the first render instead of being dropped as filtered out.
  const [navigableRows, setNavigableRows] = useState<AlertsTableRow[]>();
  const [searchParams, setSearchParams] = useSearchParams();
  const [linkNotFound, setLinkNotFound] = useState(false);
  // Node group headers are not navigable — the details pane only shows alerts, so
  // next/previous must step over them (and collapsed groups' hidden alerts drop out).
  const navigableAlerts = useMemo(
    () =>
      (navigableRows ?? rows).filter(
        (row): row is AlertRow => row.type === 'alert'
      ),
    [navigableRows, rows]
  );
  const linkedAlert = useMemo(
    () => findLinkedAlert(rows, searchParams),
    [rows, searchParams]
  );
  const selectedRow = navigableAlerts.some((row) => row.id === linkedAlert?.id)
    ? linkedAlert
    : undefined;
  const hasLink = searchParams.has(ALERT_LINK_PARAMS.rule);
  const tab: AlertDetailsPaneTab =
    searchParams.get(ALERT_LINK_PARAMS.tab) === 'raw-data'
      ? 'raw-data'
      : 'details';

  const setLink = (alert?: AlertRow, replace = false) =>
    setSearchParams((prev) => withAlertLink(prev, alert), { replace });

  const changeTab = (newTab: AlertDetailsPaneTab) =>
    setSearchParams(
      (prev) => {
        const next = new URLSearchParams(prev);

        if (newTab === 'details') {
          next.delete(ALERT_LINK_PARAMS.tab);
        } else {
          next.set(ALERT_LINK_PARAMS.tab, newTab);
        }

        return next;
      },
      { replace: true }
    );

  // Drop a link to an alert that is gone, or that the table now hides.
  useEffect(() => {
    if (!hasLink || !data || selectedRow) {
      return;
    }

    if (!linkedAlert) {
      setLinkNotFound(true);
    }

    setSearchParams((prev) => withAlertLink(prev), { replace: true });
  }, [hasLink, data, selectedRow, linkedAlert, setSearchParams]);

  const detailsPaneProps = useDetailsPaneNavigation<AlertRow>({
    rows: navigableAlerts,
    selected: selectedRow,
    getRowId: (row) => row.id,
    // Next/previous replace the history entry, so Back closes the pane.
    onSelect: (row) => setLink(row, true),
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
        p: 2,
        minHeight: 0,
        // The shell grows with its content, so cap the page at the viewport
        // to keep the table as the only scroll area.
        maxHeight: '100vh',
      }}
    >
      <Typography variant="h3">{Messages.title}</Typography>
      {linkNotFound && (
        <Alert severity="info" onClose={() => setLinkNotFound(false)}>
          {Messages.linkNotFound}
        </Alert>
      )}
      <Stack
        sx={{
          flex: 1,
          minHeight: 0,
          position: 'relative',
          // Clips the details pane, which Slide parks below this area while closed.
          overflow: 'hidden',
        }}
      >
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
              minHeight: 0,

              '& > *': {
                flex: 1,
                minHeight: 0,
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
              onOpenDetail={(row) => {
                setLinkNotFound(false);
                setLink(row);
              }}
            />
          </Stack>
        )}
        <AlertDetailsPane
          alert={selectedRow}
          tab={tab}
          onTabChange={changeTab}
          onClose={() => setLink()}
          {...detailsPaneProps}
        />
      </Stack>
    </Stack>
  );
};

export default AlertsPage;
