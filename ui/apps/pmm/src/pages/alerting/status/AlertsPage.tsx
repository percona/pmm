import { useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Skeleton,
  Stack,
  tableContainerClasses,
  Typography,
} from '@mui/material';
import { usePrometheusAlertRules } from 'hooks/api/usePrometheusAlertRules';
import { findAlertTableRowById, flattenAlertRules } from './AlertsPage.utils';
import { AlertStatusTable } from './table';
import { useDetailsPaneNavigation } from '@percona/percona-ui';
import { AlertsTableRow } from './AlertsPage.types';
import { AlertDetailsPane } from './details-pane';
import { Messages } from './AlertsPage.messages';

const AlertsPage = () => {
  const { data, isError, isLoading } = usePrometheusAlertRules({
    refetchInterval: 5000,
  });
  const rows = useMemo(() => flattenAlertRules(data), [data]);
  const [navigableRows, setNavigableRows] = useState<AlertsTableRow[]>(rows);
  const [selectedRowId, setSelectedRowId] = useState<string>();
  const selectedNavigableRow = useMemo(
    () => findAlertTableRowById(navigableRows, selectedRowId),
    [navigableRows, selectedRowId]
  );
  const selectedRow = useMemo(
    () =>
      selectedNavigableRow
        ? rows.find((row) => row.id === selectedRowId)
        : undefined,
    [rows, selectedNavigableRow, selectedRowId]
  );

  useEffect(() => {
    if (selectedRowId && (!selectedNavigableRow || !selectedRow)) {
      setSelectedRowId(undefined);
    }
  }, [selectedNavigableRow, selectedRow, selectedRowId]);

  const detailsPaneProps = useDetailsPaneNavigation<AlertsTableRow>({
    rows: navigableRows,
    selected: selectedRow,
    getRowId: (row) => row.id,
    onSelect: (row) => setSelectedRowId(row?.id),
  });

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
        ) : rows.length === 0 ? (
          <Alert severity="info">{Messages.empty}</Alert>
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
