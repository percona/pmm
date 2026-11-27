import type { FC } from 'react';
import { ServiceType } from 'types/services.types';
import { technologyLabel } from './Technology.utils';

interface Props {
  serviceType?: ServiceType;
}

// For rows that state their technology alone; the pickers use group headers instead.
const Technology: FC<Props> = ({ serviceType }) => {
  const label = technologyLabel(serviceType);

  if (!label) {
    return null;
  }

  return <span data-testid="technology">{label}</span>;
};

export default Technology;
