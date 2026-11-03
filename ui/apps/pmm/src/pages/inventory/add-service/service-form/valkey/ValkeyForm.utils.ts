import type { AddServicePayload } from 'types/services.types';
import type { ServiceNode } from '../ServiceForm.types';
import { toDatabaseService, toTlsCertificates } from '../ServiceForm.utils';
import type { ValkeyFormValues } from './ValkeyForm.types';

export const toValkeyPayload = (
  values: ValkeyFormValues,
  nodes: ServiceNode[]
): AddServicePayload => ({
  valkey: {
    ...toDatabaseService(values, nodes),
    ...toTlsCertificates(values),
  },
});
