import Stack from '@mui/material/Stack';
import type { FC } from 'react';
import type { ManagedServiceType } from 'types/services.types';
import { SERVICE_TYPE_ICONS } from '../../Inventory.constants';

interface Props {
  name: string;
  serviceType?: ManagedServiceType;
}

export const ServiceName: FC<Props> = ({ name, serviceType }) => {
  const Icon = serviceType && SERVICE_TYPE_ICONS[serviceType];

  return (
    <Stack direction="row" alignItems="center" gap={1}>
      {Icon && <Icon fontSize="small" data-testid="service-icon" />}
      <span>{name}</span>
    </Stack>
  );
};
