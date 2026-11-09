import Grid from '@mui/material/Grid';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import DataPoint from 'components/details-pane/DataPoint';
import type { FC } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import { AgentsStatusChip } from '../../components/agents-status-chip';
import { LabelsList } from '../../components/labels-list';
import { Messages as InventoryMessages } from '../../Inventory.messages';
import { getFilteredTabUrl, toLabelTags } from '../../Inventory.utils';
import { Messages } from '../Nodes.messages';
import type { NodeRow } from '../Nodes.types';

interface Props {
  node: NodeRow;
}

export const NodeDetails: FC<Props> = ({ node }) => (
  <Grid
    container
    spacing={3}
    columns={{ xs: 1, md: 3 }}
    width="100%"
    data-testid="node-details"
  >
    {!!node.agents.length && (
      <DataPoint size={1} title={InventoryMessages.details.agents}>
        <AgentsStatusChip agents={node.agents} tab="nodes" id={node.nodeId} />
      </DataPoint>
    )}
    <DataPoint size={1} title={Messages.details.nodeId}>
      {node.nodeId}
    </DataPoint>
    {!!node.services.length && (
      <DataPoint size={1} title={Messages.details.serviceNames}>
        <Stack>
          {node.services.map(({ serviceId, serviceName }) => (
            <Link
              key={serviceId}
              component={RouterLink}
              to={getFilteredTabUrl('services', 'serviceId', serviceId)}
            >
              {serviceName}
            </Link>
          ))}
        </Stack>
      </DataPoint>
    )}
    {!!toLabelTags(node.customLabels).length && (
      <DataPoint
        size={{ xs: 1, md: 3 }}
        title={InventoryMessages.details.labels}
      >
        <LabelsList labels={node.customLabels} />
      </DataPoint>
    )}
    {!!toLabelTags(node.properties).length && (
      <DataPoint size={{ xs: 1, md: 3 }} title={Messages.details.properties}>
        <LabelsList labels={node.properties} />
      </DataPoint>
    )}
  </Grid>
);
