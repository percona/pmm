import type { FC } from 'react';
import { ServiceType } from 'types/services.types';
import { technologyLabel } from './Technology.utils';

interface Props {
  serviceType?: ServiceType;
}

// Technology names the database technology of a service in words. The pickers
// carry it in their group headers instead, so this is only used where a row
// needs to state it on its own.
const Technology: FC<Props> = ({ serviceType }) => {
  const label = technologyLabel(serviceType);

  if (!label) {
    return null;
  }

  return <span data-testid="technology">{label}</span>;
};

export default Technology;
