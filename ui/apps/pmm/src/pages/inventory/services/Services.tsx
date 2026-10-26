import AddOutlinedIcon from '@mui/icons-material/AddOutlined';
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import Button from '@mui/material/Button';
import FormControlLabel from '@mui/material/FormControlLabel';
import ListItemIcon from '@mui/material/ListItemIcon';
import ListItemText from '@mui/material/ListItemText';
import MenuItem from '@mui/material/MenuItem';
import Stack from '@mui/material/Stack';
import Switch from '@mui/material/Switch';
import Typography from '@mui/material/Typography';
import EditOutlinedIcon from '@mui/icons-material/EditOutlined';
import {
  hasActiveColumnFilters,
  Table,
  type MRT_RowSelectionState,
} from '@percona/peak-ui';
import { type MRT_ExpandedState } from 'material-react-table';
import { ExportButton } from 'components/export-button';
import { useManagedServices, useRemoveServices } from 'hooks/api/useServices';
import { useLocalStorage } from 'hooks/utils/useLocalStorage';
import { useTableUrlState } from 'hooks/utils/useTableUrlState';
import { PMM_ADD_SERVICE_URL } from 'lib/constants';
import { enqueueSnackbar } from 'notistack';
import { type FC, useEffect, useMemo, useState } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import { DeleteModal } from '../components/delete-modal';
import {
  CLUSTERS_SWITCH_KEY,
  getUrlStateOptions,
  INVENTORY_REFETCH_INTERVAL,
  INVENTORY_TABLE_OPTIONS,
} from '../Inventory.constants';
import { Messages as InventoryMessages } from '../Inventory.messages';
import {
  getEditServiceUrl,
  getServiceDashboardUrl,
  getServiceQanUrl,
} from '../Inventory.utils';
import { useIdColumnVisibility } from '../useIdColumnVisibility';
import { exportServicesToCsv } from './export/exportServicesToCsv';
import { ServiceDetails } from './service-details';
import { SERVICES_COLUMNS, SERVICES_TABLE_NAME } from './Services.constants';
import { Messages } from './Services.messages';
import type { ServiceRow } from './Services.types';
import {
  getFilteredServices,
  getServicesRowId,
  isClusterRow,
  toClusterRows,
  toServiceRows,
} from './Services.utils';

export const Services: FC = () => {
  const { data, isLoading } = useManagedServices(
    {},
    { refetchInterval: INVENTORY_REFETCH_INTERVAL }
  );
  const { mutateAsync: removeServices } = useRemoveServices();
  const [showClusters, setShowClusters] = useLocalStorage(
    CLUSTERS_SWITCH_KEY,
    false
  );
  const [rowSelection, setRowSelection] = useState<MRT_RowSelectionState>({});
  const [expanded, setExpanded] = useState<MRT_ExpandedState>({});
  const [servicesToDelete, setServicesToDelete] = useState<ServiceRow[]>([]);
  const services = useMemo(() => toServiceRows(data?.services ?? []), [data]);
  const clusters = useMemo(() => toClusterRows(services), [services]);
  const selectedServices = services.filter(
    (service) => rowSelection[service.serviceId]
  );
  const { tableProps } = useTableUrlState({
    ...getUrlStateOptions('services'),
    additionalState: { rowSelection, expanded },
  });
  const { columnFilters, globalFilter } = tableProps.state;
  const { columnVisibility, onColumnVisibilityChange } = useIdColumnVisibility(
    SERVICES_TABLE_NAME,
    'serviceId',
    tableProps
  );
  const isFiltered = hasActiveColumnFilters(columnFilters) || !!globalFilter;
  const clusterIds = JSON.stringify(clusters.map(({ clusterId }) => clusterId));

  // A filtered cluster view would otherwise hide its matches in collapsed clusters
  useEffect(() => {
    if (showClusters && isFiltered) {
      setExpanded(
        Object.fromEntries(
          (JSON.parse(clusterIds) as string[]).map((id) => [id, true])
        )
      );
    }
  }, [showClusters, isFiltered, clusterIds]);

  const handleDelete = async (force: boolean) => {
    const serviceNames = servicesToDelete.map(({ serviceName }) => serviceName);
    const deleted = await removeServices(
      servicesToDelete.map(({ serviceId }) => ({ serviceId, force }))
    );

    if (deleted) {
      enqueueSnackbar(Messages.delete.success(deleted, serviceNames), {
        variant: 'success',
      });
    }

    setRowSelection({});
    setServicesToDelete([]);
  };

  return (
    <>
      <Table
        {...INVENTORY_TABLE_OPTIONS}
        {...tableProps}
        state={{
          ...tableProps.state,
          isLoading,
          columnVisibility: { ...columnVisibility, cluster: false },
        }}
        tableName={SERVICES_TABLE_NAME}
        onColumnVisibilityChange={onColumnVisibilityChange}
        columns={SERVICES_COLUMNS}
        data={showClusters ? clusters : services}
        getRowId={getServicesRowId}
        enableExpanding
        getSubRows={(row) => (isClusterRow(row) ? row.services : undefined)}
        filterFromLeafRows
        noDataMessage={Messages.empty}
        enableRowSelection
        onRowSelectionChange={setRowSelection}
        onExpandedChange={setExpanded}
        renderDetailPanel={({ row: { original } }) =>
          isClusterRow(original) ? null : <ServiceDetails service={original} />
        }
        enableRowActions
        renderRowActionMenuItems={({ row: { original }, closeMenu }) => {
          if (isClusterRow(original)) {
            return [];
          }

          const dashboardUrl = getServiceDashboardUrl(original);
          const qanUrl = getServiceQanUrl(original);

          return [
            <MenuItem
              key="delete"
              onClick={() => {
                setServicesToDelete([original]);
                closeMenu();
              }}
            >
              <ListItemIcon>
                <DeleteOutlineIcon />
              </ListItemIcon>
              <ListItemText>{InventoryMessages.actions.delete}</ListItemText>
            </MenuItem>,
            <MenuItem
              key="edit"
              component={RouterLink}
              to={getEditServiceUrl(original.serviceId)}
            >
              <ListItemIcon>
                <EditOutlinedIcon />
              </ListItemIcon>
              <ListItemText>{Messages.actions.edit}</ListItemText>
            </MenuItem>,
            ...(dashboardUrl
              ? [
                  <MenuItem
                    key="dashboard"
                    component={RouterLink}
                    to={dashboardUrl}
                  >
                    <ListItemText inset>
                      {Messages.actions.dashboard}
                    </ListItemText>
                  </MenuItem>,
                ]
              : []),
            ...(qanUrl
              ? [
                  <MenuItem key="qan" component={RouterLink} to={qanUrl}>
                    <ListItemText inset>{Messages.actions.qan}</ListItemText>
                  </MenuItem>,
                ]
              : []),
          ];
        }}
        renderTopToolbarCustomActions={({ table }) => {
          // A selection narrows the export, as it does the delete
          const exportedServices = selectedServices.length
            ? selectedServices
            : getFilteredServices(table);

          return (
            <Stack
              direction="row"
              alignItems="center"
              flexWrap="wrap"
              columnGap={2}
              rowGap={1}
            >
              {!!selectedServices.length && (
                <Typography variant="body2">
                  {InventoryMessages.selected(selectedServices.length)}
                </Typography>
              )}
              <Button
                color="error"
                startIcon={<DeleteOutlineIcon />}
                disabled={!selectedServices.length}
                onClick={() => setServicesToDelete(selectedServices)}
                data-testid="delete-services-button"
              >
                {InventoryMessages.actions.delete}
              </Button>
              <Stack direction="row" alignItems="center">
                <FormControlLabel
                  label={Messages.organizeByClusters}
                  sx={{ mr: 1 }}
                  control={
                    <Switch
                      size="small"
                      checked={showClusters}
                      onChange={() => setShowClusters(!showClusters)}
                      data-testid="organize-by-clusters-switch"
                    />
                  }
                />
              </Stack>
              <ExportButton
                disabled={!exportedServices.length}
                onClick={() => exportServicesToCsv(exportedServices)}
                data-testid="export-services-button"
              />
              <Button
                startIcon={<AddOutlinedIcon />}
                component={RouterLink}
                to={PMM_ADD_SERVICE_URL}
                data-testid="add-service-button"
              >
                {Messages.actions.add}
              </Button>
            </Stack>
          );
        }}
      />
      <DeleteModal
        open={!!servicesToDelete.length}
        title={Messages.delete.title}
        warning={Messages.delete.warning}
        message={Messages.delete.message(
          servicesToDelete.map(({ serviceName }) => serviceName)
        )}
        forceDescription={Messages.delete.forceDescription}
        confirmLabel={Messages.delete.confirm(servicesToDelete.length)}
        onConfirm={handleDelete}
        onClose={() => setServicesToDelete([])}
      />
    </>
  );
};
