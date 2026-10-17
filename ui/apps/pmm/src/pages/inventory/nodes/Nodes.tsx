import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import Button from '@mui/material/Button';
import FormControlLabel from '@mui/material/FormControlLabel';
import ListItemIcon from '@mui/material/ListItemIcon';
import ListItemText from '@mui/material/ListItemText';
import MenuItem from '@mui/material/MenuItem';
import Stack from '@mui/material/Stack';
import Switch from '@mui/material/Switch';
import Typography from '@mui/material/Typography';
import { Table, type MRT_RowSelectionState } from '@percona/peak-ui';
import AlertThresholds from 'components/alert-thresholds';
import type { AlertThresholdsNode } from 'components/alert-thresholds/AlertThresholds.types';
import { useHaInfo } from 'hooks/api/useHA';
import { useNodes, useRemoveNodes } from 'hooks/api/useNodes';
import { useTableUrlState } from 'hooks/utils/useTableUrlState';
import { enqueueSnackbar } from 'notistack';
import { type FC, useMemo, useState } from 'react';
import { DeleteModal } from '../components/delete-modal';
import {
  getUrlStateOptions,
  INVENTORY_REFETCH_INTERVAL,
  INVENTORY_TABLE_OPTIONS,
} from '../Inventory.constants';
import { Messages as InventoryMessages } from '../Inventory.messages';
import { useIdColumnVisibility } from '../useIdColumnVisibility';
import { NodeDetails } from './node-details';
import { NODES_COLUMNS } from './Nodes.constants';
import { Messages } from './Nodes.messages';
import type { NodeRow } from './Nodes.types';
import { canRemoveNode, toNodeRows } from './Nodes.utils';

const NODES_TABLE_NAME = 'inventory-nodes';

const PMM_SERVER_NODE_FILTER = { id: 'isPmmServerNode', value: 'true' };

export const Nodes: FC = () => {
  const { data, isLoading } = useNodes({
    refetchInterval: INVENTORY_REFETCH_INTERVAL,
  });
  const { data: ha } = useHaInfo();
  const { mutateAsync: removeNodes } = useRemoveNodes();
  const [rowSelection, setRowSelection] = useState<MRT_RowSelectionState>({});
  const [nodesToDelete, setNodesToDelete] = useState<NodeRow[]>([]);
  const [thresholdsNode, setThresholdsNode] =
    useState<AlertThresholdsNode | null>(null);
  const nodes = useMemo(
    () => toNodeRows(data ?? [], ha.nodes),
    [data, ha.nodes]
  );
  const selectedNodes = nodes.filter(({ nodeId }) => rowSelection[nodeId]);
  const { tableProps } = useTableUrlState({
    ...getUrlStateOptions('nodes'),
    additionalState: { rowSelection },
  });
  const { columnFilters } = tableProps.state;
  const { columnVisibility, onColumnVisibilityChange } = useIdColumnVisibility(
    NODES_TABLE_NAME,
    'nodeId',
    tableProps
  );
  const pmmServerOnly = columnFilters.some(
    ({ id, value }) =>
      id === PMM_SERVER_NODE_FILTER.id && value === PMM_SERVER_NODE_FILTER.value
  );

  const togglePmmServerOnly = () =>
    tableProps.onColumnFiltersChange((filters) => [
      ...filters.filter(({ id }) => id !== PMM_SERVER_NODE_FILTER.id),
      ...(pmmServerOnly ? [] : [PMM_SERVER_NODE_FILTER]),
    ]);

  const handleDelete = async (force: boolean) => {
    const { deleted, warnings } = await removeNodes(
      nodesToDelete.map(({ nodeId }) => ({ nodeId, force }))
    );

    if (deleted) {
      enqueueSnackbar(Messages.delete.success(deleted, nodesToDelete.length), {
        variant: 'success',
      });
    }

    warnings.forEach((warning) =>
      enqueueSnackbar(warning, { variant: 'warning' })
    );

    setRowSelection({});
    setNodesToDelete([]);
  };

  return (
    <>
      <Table
        {...INVENTORY_TABLE_OPTIONS}
        {...tableProps}
        state={{
          ...tableProps.state,
          isLoading,
          columnVisibility: { ...columnVisibility, isPmmServerNode: false },
        }}
        tableName={NODES_TABLE_NAME}
        onColumnVisibilityChange={onColumnVisibilityChange}
        columns={NODES_COLUMNS}
        data={nodes}
        getRowId={({ nodeId }) => nodeId}
        noDataMessage={Messages.empty}
        enableRowSelection={({ original }) => canRemoveNode(original)}
        onRowSelectionChange={setRowSelection}
        renderDetailPanel={({ row: { original } }) => (
          <NodeDetails node={original} />
        )}
        enableRowActions
        renderRowActionMenuItems={({ row: { original }, closeMenu }) => [
          <MenuItem
            key="override-thresholds"
            onClick={() => {
              setThresholdsNode({
                nodeId: original.nodeId,
                nodeName: original.nodeName,
              });
              closeMenu();
            }}
          >
            <ListItemText>{Messages.actions.overrideThresholds}</ListItemText>
          </MenuItem>,
          ...(canRemoveNode(original)
            ? [
                <MenuItem
                  key="delete"
                  onClick={() => {
                    setNodesToDelete([original]);
                    closeMenu();
                  }}
                >
                  <ListItemIcon>
                    <DeleteOutlineIcon />
                  </ListItemIcon>
                  <ListItemText>
                    {InventoryMessages.actions.delete}
                  </ListItemText>
                </MenuItem>,
              ]
            : []),
        ]}
        renderTopToolbarCustomActions={() => (
          <Stack
            direction="row"
            alignItems="center"
            flexWrap="wrap"
            columnGap={2}
            rowGap={1}
          >
            <Button
              color="error"
              startIcon={<DeleteOutlineIcon />}
              disabled={!selectedNodes.length}
              onClick={() => setNodesToDelete(selectedNodes)}
              data-testid="delete-nodes-button"
            >
              {InventoryMessages.actions.delete}
            </Button>
            {!!selectedNodes.length && (
              <Typography variant="body2">
                {InventoryMessages.selected(selectedNodes.length)}
              </Typography>
            )}
            {(ha.enabled || pmmServerOnly) && (
              <FormControlLabel
                label={Messages.pmmServerOnly}
                control={
                  <Switch
                    size="small"
                    checked={pmmServerOnly}
                    onChange={togglePmmServerOnly}
                    data-testid="pmm-server-only-switch"
                  />
                }
              />
            )}
          </Stack>
        )}
      />
      <AlertThresholds
        node={thresholdsNode}
        onClose={() => setThresholdsNode(null)}
      />
      <DeleteModal
        open={!!nodesToDelete.length}
        title={Messages.delete.title}
        message={Messages.delete.message(nodesToDelete.length)}
        forceDescription={Messages.delete.forceDescription}
        confirmLabel={Messages.delete.confirm}
        onConfirm={handleDelete}
        onClose={() => setNodesToDelete([])}
      />
    </>
  );
};
