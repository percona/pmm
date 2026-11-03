import type { AddServicePayload } from 'types/services.types';
import type { ServiceNode } from '../ServiceForm.types';
import { toCollectors, toDatabaseService } from '../ServiceForm.utils';
import type { ProxySQLFormValues } from './ProxySQLForm.types';

export const toProxySQLPayload = (
  values: ProxySQLFormValues,
  nodes: ServiceNode[]
): AddServicePayload => ({
  proxysql: {
    ...toDatabaseService(values, nodes),
    disableCollectors: toCollectors(values.disableCollectors),
  },
});
