import type { FC } from 'react';
import Box from '@mui/material/Box';
import Tooltip from '@mui/material/Tooltip';
import { MongoIcon, MySqlIcon } from '@percona/percona-ui';
import { ServiceType } from 'types/services.types';
import { technologyLabel } from './Technology.utils';

const TECHNOLOGY_ICONS: Partial<Record<ServiceType, typeof MySqlIcon>> = {
  [ServiceType.mongodb]: MongoIcon,
  [ServiceType.mysql]: MySqlIcon,
};

interface Props {
  serviceType?: ServiceType;
  // Where horizontal space is tight (the services dropdown) the name is dropped
  // and the icon carries the meaning, with the label moved into a tooltip.
  iconOnly?: boolean;
}

const Technology: FC<Props> = ({ serviceType, iconOnly = false }) => {
  const label = technologyLabel(serviceType);
  const TechnologyIcon = serviceType
    ? TECHNOLOGY_ICONS[serviceType]
    : undefined;

  if (!label) {
    return null;
  }

  const icon = TechnologyIcon ? (
    <TechnologyIcon fontSize="small" data-testid={`technology-icon-${label}`} />
  ) : null;

  if (iconOnly) {
    return icon ? (
      <Tooltip title={label} arrow>
        <Box
          sx={{ display: 'flex', alignItems: 'center' }}
          data-testid="technology"
        >
          {icon}
        </Box>
      </Tooltip>
    ) : null;
  }

  return (
    <Box
      sx={{ display: 'flex', alignItems: 'center', gap: 0.75 }}
      data-testid="technology"
    >
      {icon}
      {label}
    </Box>
  );
};

export default Technology;
