import { useEffect, useMemo, useState } from 'react';
import { format } from 'date-fns';
import { tz } from '@date-fns/tz';
import { Link as RouterLink } from 'react-router-dom';
import { Table } from '@percona/percona-ui';
import { type MRT_ColumnDef } from 'material-react-table';
import {
  Alert,
  Button,
  Card,
  CardContent,
  Chip,
  FormControl,
  FormControlLabel,
  InputLabel,
  MenuItem,
  Select,
  Skeleton,
  Stack,
  Switch,
  Typography,
} from '@mui/material';
import { paperClasses } from '@mui/material/Paper';
import { Page } from 'components/page';
import { useUser } from 'contexts/user';
import { usePrometheusAlertRules } from 'hooks/api/usePrometheusAlertRules';
import { TIME_FORMAT } from 'lib/constants';
import { AlertRow, AlertsTableRow } from './AlertsPage.types';
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
import {
  ALL_STATES_FILTER,
  STATE_OPTIONS,
  STATUS_COLOR_MAP,
  STATUS_LABEL_MAP,
} from './AlertsPage.constants';
import { AlertDetailsPane } from './details-pane';

const createAlertRuleViewUrl = (ruleGroupUid: string) =>
  `/graph/alerting/grafana/${ruleGroupUid}/view`;
const createAlertRuleEditUrl = (ruleGroupUid: string) =>
  `/graph/alerting/${ruleGroupUid}/edit`;

const formatTimestamp = (timestamp: string | undefined, timezone: string) => {
  if (!timestamp) {
    return '-';
  }

  const date = new Date(timestamp);

  if (Number.isNaN(date.getTime())) {
    return '-';
  }

  return format(date, TIME_FORMAT, { in: tz(timezone) });
};

const AlertsPage = () => {
  const { user } = useUser();
  const timezone = user?.preferences?.timezone || 'UTC';
  const { data, isLoading, isError, error, refetch, isRefetching } =
    usePrometheusAlertRules({
      refetchInterval: 5000,
    });
  const [isGroupedByNode, setIsGroupedByNode] = useState<boolean>(false);
  const [selectedNode, setSelectedNode] = useState<string>(ALL_NODES_FILTER);
  const [selectedService, setSelectedService] =
    useState<string>(ALL_SERVICES_FILTER);
  const [selectedState, setSelectedState] = useState<string>(ALL_STATES_FILTER);
  const [selectedAlert, setSelectedAlert] = useState<AlertRow>();
  const [selectedAlertIndex, setSelectedAlertIndex] = useState<number>();
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

  const filteredRows = useMemo(() => {
    const rows = isServiceFilterDisabled
      ? nodeFilteredRows
      : filterAlertRulesByService(nodeFilteredRows, selectedService);

    if (selectedState !== ALL_STATES_FILTER) {
      return rows.filter((r) => r.state === selectedState);
    }

    return rows;
  }, [
    nodeFilteredRows,
    selectedService,
    isServiceFilterDisabled,
    selectedState,
  ]);
  const tableRows = useMemo<AlertsTableRow[]>(
    () => (isGroupedByNode ? groupAlertsByNode(filteredRows) : filteredRows),
    [filteredRows, isGroupedByNode]
  );

  useEffect(() => {
    if (!selectedAlert) {
      return;
    }

    const nextIndex = filteredRows.findIndex(
      (row) => row.id === selectedAlert.id
    );

    if (nextIndex === -1) {
      setSelectedAlert(undefined);
      setSelectedAlertIndex(undefined);
      return;
    }

    if (filteredRows[nextIndex] !== selectedAlert) {
      setSelectedAlert(filteredRows[nextIndex]);
      setSelectedAlertIndex(nextIndex);
    }
  }, [filteredRows, selectedAlert]);

  const handleAlertChange = (alert: AlertRow) => {
    setSelectedAlert(alert);
    setSelectedAlertIndex(filteredRows.findIndex((row) => row.id === alert.id));
  };

  const handleCloseDetails = () => {
    setSelectedAlert(undefined);
    setSelectedAlertIndex(undefined);
  };

  const handleNextAlert = () => {
    const idx = (selectedAlertIndex ?? -1) + 1;

    if (idx >= filteredRows.length) {
      return;
    }

    handleAlertChange(filteredRows[idx]);
  };

  const handlePreviousAlert = () => {
    const idx = (selectedAlertIndex ?? 0) - 1;

    if (idx < 0) {
      return;
    }

    handleAlertChange(filteredRows[idx]);
  };

  const columns = useMemo<MRT_ColumnDef<AlertsTableRow>[]>(
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
        accessorKey: 'nodeId',
        header: 'Node',
        Cell: ({ row }) =>
          row.original.type === 'node' && isGroupedByNode
            ? '-'
            : row.original.nodeId || '-',
      },
      {
        accessorKey: 'serviceName',
        header: 'Service',
        Cell: ({ row }) =>
          row.original.type === 'node' ? '-' : row.original.serviceName || '-',
      },
      {
        accessorKey: 'activeAt',
        header: 'Active since',
        Cell: ({ row }) =>
          row.original.type === 'node'
            ? '-'
            : formatTimestamp(row.original.activeAt, timezone),
      },
      {
        accessorKey: 'age',
        header: 'Age',
        Cell: ({ row }) =>
          row.original.type === 'node' ? '-' : row.original.age,
      },
    ],
    [isGroupedByNode, timezone]
  );

  return (
    <Stack
      sx={{
        flex: 1,
        gap: 2,
        m: 2,
      }}
    >
      <Typography variant="h3">Alerts</Typography>
      <Card
        variant="outlined"
        sx={{
          position: 'relative',
          flex: 1,
          minHeight: 0,
          display: 'flex',
          flexDirection: 'column',
          overflow: 'hidden',
        }}
      >
        <CardContent
          sx={{
            flex: 1,
            minHeight: 0,
            display: 'flex',
            flexDirection: 'column',
          }}
        >
          <Stack spacing={2} sx={{ flex: 1, minHeight: 0 }}>
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
              <Stack
                spacing={1.5}
                sx={{
                  flex: 1,
                  minHeight: 0,
                  [`& > .${paperClasses.root}`]: {
                    flex: 1,
                    display: 'flex',
                    flexDirection: 'column',
                    minHeight: 0,
                    overflow: 'hidden',
                  },
                }}
              >
                <Stack
                  direction="row"
                  justifyContent="space-between"
                  alignItems="center"
                  flexWrap="wrap"
                  gap={1}
                >
                  <Stack direction="row" alignItems="center" gap={2}>
                    <FormControl sx={{ width: 200 }} size="small">
                      <InputLabel id="node">Node</InputLabel>
                      <Select
                        labelId="node"
                        label="Node"
                        value={selectedNode}
                        onChange={(e) => setSelectedNode(e.target.value)}
                      >
                        {nodeOptions.map((opt) => (
                          <MenuItem key={opt.value} value={opt.value}>
                            {opt.label}
                          </MenuItem>
                        ))}
                      </Select>
                    </FormControl>
                    <FormControl sx={{ width: 200 }} size="small">
                      <InputLabel id="service">Service</InputLabel>
                      <Select
                        labelId="service"
                        label="Service"
                        value={selectedService}
                        onChange={(e) => setSelectedService(e.target.value)}
                        disabled={isServiceFilterDisabled}
                      >
                        {serviceOptions.map((opt) => (
                          <MenuItem key={opt.value} value={opt.value}>
                            {opt.label}
                          </MenuItem>
                        ))}
                      </Select>
                    </FormControl>
                    <FormControl sx={{ width: 200 }} size="small">
                      <InputLabel id="state">State</InputLabel>
                      <Select
                        labelId="state"
                        label="State"
                        value={selectedState}
                        onChange={(e) => setSelectedState(e.target.value)}
                      >
                        {STATE_OPTIONS.map((opt) => (
                          <MenuItem key={opt.value} value={opt.value}>
                            {opt.label}
                          </MenuItem>
                        ))}
                      </Select>
                    </FormControl>
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
                      'mrt-row-actions',
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
                  enableRowActions
                  displayColumnDefOptions={{
                    'mrt-row-actions': {
                      header: '',
                      size: 48,
                      minSize: 48,
                      maxSize: 48,
                      muiTableHeadCellProps: {
                        sx: {
                          width: 48,
                          minWidth: 48,
                          px: 0.5,
                        },
                      },
                      muiTableBodyCellProps: {
                        sx: {
                          width: 48,
                          minWidth: 48,
                          px: 0.5,
                        },
                      },
                    },
                  }}
                  renderRowActionMenuItems={({ row, closeMenu }) => {
                    if (
                      row.original.type !== 'alert' ||
                      !row.original.ruleGroupUid
                    ) {
                      return [];
                    }

                    return [
                      <MenuItem
                        key="view"
                        component={RouterLink}
                        to={createAlertRuleViewUrl(row.original.ruleGroupUid)}
                        onClick={(event) => {
                          event.stopPropagation();
                          closeMenu();
                        }}
                      >
                        View rule
                      </MenuItem>,
                      <MenuItem
                        key="edit"
                        component={RouterLink}
                        to={createAlertRuleEditUrl(row.original.ruleGroupUid)}
                        onClick={(event) => {
                          event.stopPropagation();
                          closeMenu();
                        }}
                      >
                        Edit rule
                      </MenuItem>,
                    ];
                  }}
                  getRowId={(row) => row.id}
                  getSubRows={(row) =>
                    row.type === 'node' ? row.alerts : undefined
                  }
                  muiTableBodyRowProps={({ row }) => {
                    if (row.original.type === 'alert') {
                      return {
                        sx: {
                          cursor: 'pointer',
                          '& td': {
                            fontWeight: 400,
                          },
                        },
                      };
                    }

                    return {
                      sx: {
                        cursor: 'default',
                        '& td': {
                          fontWeight: 600,
                        },
                      },
                    };
                  }}
                  muiTableBodyCellProps={({ row }) => {
                    if (row.original.type !== 'alert') {
                      return {};
                    }

                    const alert = row.original;

                    return {
                      onClick: () => handleAlertChange(alert),
                    };
                  }}
                />
              </Stack>
            )}
          </Stack>
        </CardContent>
        <AlertDetailsPane
          alert={selectedAlert}
          onClose={handleCloseDetails}
          isFirstAlert={selectedAlertIndex === 0}
          isLastAlert={selectedAlertIndex === filteredRows.length - 1}
          onNext={handleNextAlert}
          onPrevious={handlePreviousAlert}
        />
      </Card>
    </Stack>
  );
};

export default AlertsPage;
