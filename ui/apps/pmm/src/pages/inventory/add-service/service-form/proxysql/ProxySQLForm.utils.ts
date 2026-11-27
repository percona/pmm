import type { AddServicePayload } from 'types/services.types';
import { toCollectors, toDatabaseService } from '../ServiceForm.utils';
import type { ProxySQLFormValues } from './ProxySQLForm.types';

export const toProxySQLPayload = (
  values: ProxySQLFormValues
): AddServicePayload => ({
  proxysql: {
    ...toDatabaseService(values),
    disableCollectors: toCollectors(values.disableCollectors),
  },
});
