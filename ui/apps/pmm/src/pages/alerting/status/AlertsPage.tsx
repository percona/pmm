import { useMemo, useState } from 'react';
import { Stack, tableContainerClasses, Typography } from '@mui/material';
import { usePrometheusAlertRules } from 'hooks/api/usePrometheusAlertRules';
import { flattenAlertRules } from './AlertsPage.utils';
import { AlertStatusTable } from './table';
import { useDetailsPaneNavigation } from '@percona/percona-ui';
import { AlertsTableRow } from './AlertsPage.types';
import { AlertDetailsPane } from './details-pane';

const AlertsPage = () => {
  const { data } = usePrometheusAlertRules({
    refetchInterval: 5000,
  });
  const rows = useMemo(() => flattenAlertRules(data), [data]);
  const [navigableRows, setNavigableRows] = useState<AlertsTableRow[]>(rows);
  const [selectedRow, setSelectedRow] = useState<AlertsTableRow>();
  const detailsPaneProps = useDetailsPaneNavigation<AlertsTableRow>({
    rows: navigableRows,
    selected: selectedRow,
    getRowId: (row) => row.id,
    onSelect: setSelectedRow,
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
      <Typography variant="h3">Alerts</Typography>
      <Stack flex="1" position="relative">
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
            onOpenDetail={setSelectedRow}
          />
        </Stack>
        <AlertDetailsPane
          alert={selectedRow}
          onClose={() => setSelectedRow(undefined)}
          {...detailsPaneProps}
        />
      </Stack>
    </Stack>
  );
};

export default AlertsPage;
