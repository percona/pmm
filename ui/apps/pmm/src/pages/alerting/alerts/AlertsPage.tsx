import { useEffect, useMemo, useState } from 'react';
import { Table } from '@percona/percona-ui';
import { type MRT_ColumnDef } from 'material-react-table';
import {
  Alert,
  Button,
  Card,
  CardContent,
  Chip,
  FormControlLabel,
  Skeleton,
  Stack,
  Switch,
} from '@mui/material';
import { Page } from 'components/page';
import { usePrometheusAlertRules } from 'hooks/api/usePrometheusAlertRules';
import { TextSelect } from 'components/text-select';
import { AlertsTableRow } from './AlertsPage.types';
import {
  ALL_SERVICES_FILTER,
  ALL_NODES_FILTER,
  filterAlertRulesByNode,
  filterAlertRulesByService,
  flattenAlertRules,
  getServiceFilterOptionsForNode,
  groupAlertsByNode,
  getNodeFilterOptions,
} from './AlertsPage.utils';
import { AlertStatus } from 'types/alerting.types';

const STATUS_COLOR_MAP: Record<
  AlertStatus,
  'default' | 'error' | 'warning' | 'success'
> = {
  Alerting: 'error',
  Pending: 'warning',
  Normal: 'success',
  NoData: 'default',
  Error: 'error',
};

const STATUS_LABEL_MAP: Record<AlertStatus, string> = {
  Alerting: 'Firing',
  Pending: 'Pending',
  Normal: 'OK',
  NoData: 'No Data',
  Error: 'Error',
};

const AlertsPage = () => {
  const { data, isLoading, isError, error, refetch, isRefetching } =
    usePrometheusAlertRules({
      refetchInterval: 5000,
    });
  const [isGroupedByNode, setIsGroupedByNode] = useState<boolean>(false);
  const [selectedNode, setSelectedNode] = useState<string>(ALL_NODES_FILTER);
  const [selectedService, setSelectedService] =
    useState<string>(ALL_SERVICES_FILTER);
  const rows = useMemo(() => flattenAlertRules(data), [data]);
  const nodeOptions = useMemo(() => getNodeFilterOptions(rows), [rows]);
  const nodeFilteredRows = useMemo(
    () => filterAlertRulesByNode(rows, selectedNode),
    [rows, selectedNode]
  );
  const isServiceFilterDisabled = selectedNode === ALL_NODES_FILTER;
  const serviceOptions = useMemo(
    () =>
      isServiceFilterDisabled
        ? [{ value: ALL_SERVICES_FILTER, label: 'All services' }]
        : getServiceFilterOptionsForNode(rows, selectedNode),
    [rows, selectedNode, isServiceFilterDisabled]
  );

  useEffect(() => {
    if (isServiceFilterDisabled && selectedService !== ALL_SERVICES_FILTER) {
      setSelectedService(ALL_SERVICES_FILTER);
      return;
    }

    if (!isServiceFilterDisabled) {
      const selectedServiceExists = serviceOptions.some(
        (option) => option.value === selectedService
      );

      if (!selectedServiceExists) {
        setSelectedService(ALL_SERVICES_FILTER);
      }
    }
  }, [isServiceFilterDisabled, selectedService, serviceOptions]);

  const filteredRows = useMemo(
    () =>
      isServiceFilterDisabled
        ? nodeFilteredRows
        : filterAlertRulesByService(nodeFilteredRows, selectedService),
    [nodeFilteredRows, selectedService, isServiceFilterDisabled]
  );
  const tableRows = useMemo<AlertsTableRow[]>(
    () => (isGroupedByNode ? groupAlertsByNode(filteredRows) : filteredRows),
    [filteredRows, isGroupedByNode]
  );

  const columns: MRT_ColumnDef<AlertsTableRow>[] = useMemo(
    () => [
      {
        accessorKey: 'state',
        header: 'State',
        size: 120,
        Cell: ({ row }) => {
          if (row.original.type === 'node') {
            return '';
          }

          const status = row.original.state;

          return (
            <Chip
              label={STATUS_LABEL_MAP[status]}
              color={STATUS_COLOR_MAP[status]}
              size="small"
            />
          );
        },
      },
      {
        accessorKey: 'alertName',
        header: 'Alert',
        Cell: ({ row }) =>
          row.original.type === 'node'
            ? `${row.original.nodeId} (${row.original.alertCount})`
            : row.original.alertName,
      },
      {
        accessorKey: 'ruleName',
        header: 'Rule',
        Cell: ({ row }) =>
          row.original.type === 'node' ? '-' : row.original.ruleName,
      },
      {
        accessorKey: 'nodeId',
        header: 'Node',
        Cell: ({ row }) => row.original.nodeId || '-',
      },
      {
        accessorKey: 'serviceName',
        header: 'Service',
        Cell: ({ row }) =>
          row.original.type === 'node' ? '-' : row.original.serviceName || '-',
      },
      {
        accessorKey: 'age',
        header: 'Age',
        Cell: ({ row }) =>
          row.original.type === 'node' ? '-' : row.original.age,
      },
    ],
    []
  );

  return (
    <Page title="Alerts">
      <Card variant="outlined">
        <CardContent>
          <Stack spacing={2}>
            {isLoading && <Skeleton variant="rounded" height={380} />}
            {isError && (
              <Alert
                severity="error"
                action={
                  <Button
                    color="inherit"
                    size="small"
                    onClick={() => refetch()}
                    disabled={isRefetching}
                  >
                    Retry
                  </Button>
                }
              >
                Failed to load alert rules: {error?.message || 'unknown error'}
              </Alert>
            )}
            {!isLoading && !isError && rows.length === 0 && (
              <Alert severity="info">
                No alerts were returned by Prometheus alert rules.
              </Alert>
            )}
            {!isLoading && !isError && rows.length > 0 && (
              <Stack spacing={1.5}>
                <Stack
                  direction="row"
                  justifyContent="space-between"
                  alignItems="center"
                  flexWrap="wrap"
                  gap={1}
                >
                  <Stack direction="row" alignItems="center" gap={2}>
                    <FormControlLabel
                      control={
                        <Switch
                          checked={isGroupedByNode}
                          onChange={(_, checked) => setIsGroupedByNode(checked)}
                        />
                      }
                      label="Group by node"
                      sx={{ ml: 0 }}
                    />
                  </Stack>
                  <Stack direction="row" alignItems="center" gap={1}>
                    <TextSelect
                      label="Node"
                      value={selectedNode}
                      options={nodeOptions}
                      onChange={setSelectedNode}
                      data-testid-button="alerts-node-filter"
                    />
                    <TextSelect
                      label="Service"
                      value={selectedService}
                      options={serviceOptions}
                      onChange={setSelectedService}
                      disabled={isServiceFilterDisabled}
                      disabledValue="All selected"
                      data-testid-button="alerts-service-filter"
                    />
                  </Stack>
                </Stack>
                <Table
                  initialState={{
                    pagination: {
                      pageSize: 25,
                      pageIndex: 0,
                    },
                    columnVisibility: {
                      id: true,
                    },
                    columnOrder: [
                      'mrt-row-expand',
                      ...columns.map((column) => column.accessorKey || ''),
                    ],
                  }}
                  tableName="alerts"
                  columns={columns}
                  data={tableRows}
                  noDataMessage="No alerts for selected filters."
                  enableHiding={false}
                  enableGlobalFilter={false}
                  enableFilters={false}
                  enableStickyHeader
                  enableExpanding={isGroupedByNode}
                  enableExpandAll={isGroupedByNode}
                  enableColumnActions={false}
                  enableColumnOrdering={false}
                  enableColumnDragging={false}
                  enableTopToolbar={false}
                  getRowId={(row) => row.id}
                  getSubRows={(row) =>
                    row.type === 'node' ? row.alerts : undefined
                  }
                  muiTableBodyRowProps={({ row }) => ({
                    sx: {
                      '& td': {
                        fontWeight: row.original.type === 'node' ? 600 : 400,
                      },
                    },
                  })}
                />
              </Stack>
            )}
          </Stack>
        </CardContent>
      </Card>
    </Page>
  );
};

export default AlertsPage;
