import Grid from '@mui/material/Grid';
import DataPoint from 'components/details-pane/DataPoint';
import type { FC } from 'react';
import { AgentsStatusChip } from '../../components/agents-status-chip';
import { LabelsList } from '../../components/labels-list';
import { Messages as InventoryMessages } from '../../Inventory.messages';
import { toLabelTags } from '../../Inventory.utils';
import { Messages } from '../Services.messages';
import type { ServiceRow } from '../Services.types';

interface Props {
  service: ServiceRow;
}

export const ServiceDetails: FC<Props> = ({ service }) => (
  <Grid
    container
    spacing={3}
    columns={{ xs: 1, md: 3 }}
    width="100%"
    data-testid="service-details"
  >
    {!!service.agents.length && (
      <DataPoint size={1} title={InventoryMessages.details.agents}>
        <AgentsStatusChip
          agents={service.agents}
          tab="services"
          id={service.serviceId}
        />
      </DataPoint>
    )}
    <DataPoint size={1} title={Messages.details.serviceId}>
      {service.serviceId}
    </DataPoint>
    {!!toLabelTags(service.labels).length && (
      <DataPoint
        size={{ xs: 1, md: 3 }}
        title={InventoryMessages.details.labels}
      >
        <LabelsList labels={service.labels} />
      </DataPoint>
    )}
  </Grid>
);
