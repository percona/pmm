import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import type { FC } from 'react';
import type { InventoryService } from 'types/services.types';
import { Messages } from '../EditService.messages';
import { SectionHeader } from '../section-header';

const { details } = Messages;

const getEndpoint = ({ address, port, socket }: InventoryService) => {
  if (socket) {
    return { label: details.socket, value: socket };
  }

  if (address) {
    return {
      label: details.address,
      value: port ? `${address}:${port}` : address,
    };
  }

  return undefined;
};

interface Props {
  service: InventoryService;
  nodeName: string;
}

export const ServiceDetails: FC<Props> = ({ service, nodeName }) => {
  const endpoint = getEndpoint(service);
  const items = [
    { label: details.type, value: details.types[service.serviceType] },
    { label: details.node, value: nodeName },
    ...(endpoint ? [endpoint] : []),
  ];

  return (
    <Stack gap={2} data-testid="service-details">
      <SectionHeader title={details.title} description={details.description} />
      <Stack direction={{ xs: 'column', md: 'row' }} gap={{ xs: 1, md: 4 }}>
        {items.map(({ label, value }) => (
          <Stack key={label} gap={0.5}>
            <Typography variant="body2" color="text.secondary">
              {label}
            </Typography>
            <Typography sx={{ wordBreak: 'break-all' }}>{value}</Typography>
          </Stack>
        ))}
      </Stack>
    </Stack>
  );
};
