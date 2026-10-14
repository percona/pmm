import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import Button from '@mui/material/Button';
import Grid from '@mui/material/Grid';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { Table, type MRT_RowSelectionState } from '@percona/peak-ui';
import { Page } from 'components/page';
import DataPoint from 'components/details-pane/DataPoint';
import { useAgents, useRemoveAgents } from 'hooks/api/useAgents';
import { useNodes } from 'hooks/api/useNodes';
import { useManagedServices } from 'hooks/api/useServices';
import { useTableUrlState } from 'hooks/utils/useTableUrlState';
import {
  PMM_INVENTORY_NODES_PATH,
  PMM_INVENTORY_SERVICES_PATH,
} from 'lib/constants';
import { enqueueSnackbar } from 'notistack';
import { type FC, useState } from 'react';
import { Link as RouterLink, useParams } from 'react-router-dom';
import type { ManagedAgent } from 'types/agent.types';
import { OrgRole } from 'types/user.types';
import { DeleteModal } from '../components/delete-modal';
import { LabelsList } from '../components/labels-list';
import {
  getUrlStateOptions,
  INVENTORY_REFETCH_INTERVAL,
  INVENTORY_TABLE_OPTIONS,
} from '../Inventory.constants';
import { Messages as InventoryMessages } from '../Inventory.messages';
import { toLabelTags } from '../Inventory.utils';
import { AGENTS_COLUMNS } from './Agents.constants';
import { Messages } from './Agents.messages';

export const Agents: FC = () => {
  const { serviceId, nodeId } = useParams();
  const { data: agents = [], isLoading } = useAgents(
    { serviceId, nodeId },
    { refetchInterval: INVENTORY_REFETCH_INTERVAL }
  );
  const { data: servicesData } = useManagedServices(
    {},
    { enabled: !!serviceId }
  );
  const { data: nodes } = useNodes({ enabled: !!nodeId });
  const { mutateAsync: removeAgents } = useRemoveAgents();
  const [rowSelection, setRowSelection] = useState<MRT_RowSelectionState>({});
  const [agentsToDelete, setAgentsToDelete] = useState<ManagedAgent[]>([]);
  const selectedAgents = agents.filter(({ agentId }) => rowSelection[agentId]);
  const { tableProps } = useTableUrlState({
    ...getUrlStateOptions('agents'),
    additionalState: { rowSelection },
  });
  const service = servicesData?.services.find(
    (service) => service.serviceId === serviceId
  );
  const node = nodes?.find((node) => node.nodeId === nodeId);
  const title = service
    ? Messages.serviceTitle(service.serviceName)
    : node && Messages.nodeTitle(node.nodeName);

  const handleDelete = async (force: boolean) => {
    const deleted = await removeAgents(
      agentsToDelete.map(({ agentId }) => ({ agentId, force }))
    );

    if (deleted) {
      enqueueSnackbar(Messages.delete.success(deleted, agentsToDelete.length), {
        variant: 'success',
      });
    }

    setRowSelection({});
    setAgentsToDelete([]);
  };

  return (
    <Page
      title={InventoryMessages.title}
      maxWidth="full"
      roles={[OrgRole.Admin]}
    >
      <Stack gap={2} sx={{ flex: 1 }}>
        <Link
          component={RouterLink}
          to={
            serviceId ? PMM_INVENTORY_SERVICES_PATH : PMM_INVENTORY_NODES_PATH
          }
          sx={{ display: 'flex', alignItems: 'center', gap: 1 }}
          data-testid="agents-go-back"
        >
          <ArrowBackIcon fontSize="small" />
          {serviceId ? Messages.goBackToServices : Messages.goBackToNodes}
        </Link>
        {title && (
          <Typography variant="h5" noWrap data-testid="agents-title">
            {title}
          </Typography>
        )}
        <Table
          {...INVENTORY_TABLE_OPTIONS}
          {...tableProps}
          state={{ ...tableProps.state, isLoading }}
          tableName="inventory-agents"
          columns={AGENTS_COLUMNS}
          data={agents}
          getRowId={({ agentId }) => agentId}
          noDataMessage={Messages.empty}
          enableRowSelection
          onRowSelectionChange={setRowSelection}
          renderDetailPanel={({ row: { original } }) =>
            toLabelTags(original.properties).length ? (
              <Grid container width="100%" data-testid="agent-details">
                <DataPoint size={12} title={Messages.details.properties}>
                  <LabelsList labels={original.properties} />
                </DataPoint>
              </Grid>
            ) : null
          }
          renderTopToolbarCustomActions={() => (
            <Stack direction="row" alignItems="center" gap={2}>
              <Button
                color="error"
                startIcon={<DeleteOutlineIcon />}
                disabled={!selectedAgents.length}
                onClick={() => setAgentsToDelete(selectedAgents)}
                data-testid="delete-agents-button"
              >
                {InventoryMessages.actions.delete}
              </Button>
              {!!selectedAgents.length && (
                <Typography variant="body2">
                  {InventoryMessages.selected(selectedAgents.length)}
                </Typography>
              )}
            </Stack>
          )}
        />
      </Stack>
      <DeleteModal
        open={!!agentsToDelete.length}
        title={Messages.delete.title}
        message={Messages.delete.message(agentsToDelete.length)}
        forceDescription={Messages.delete.forceDescription}
        confirmLabel={Messages.delete.confirm}
        onConfirm={handleDelete}
        onClose={() => setAgentsToDelete([])}
      />
    </Page>
  );
};
