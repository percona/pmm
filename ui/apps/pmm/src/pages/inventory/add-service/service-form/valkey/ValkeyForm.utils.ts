import type { AddServicePayload } from 'types/services.types';
import { toDatabaseService, toTlsCertificates } from '../ServiceForm.utils';
import type { ValkeyFormValues } from './ValkeyForm.types';

export const toValkeyPayload = (
  values: ValkeyFormValues
): AddServicePayload => ({
  valkey: {
    ...toDatabaseService(values),
    ...toTlsCertificates(values),
  },
});
