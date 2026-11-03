import type { AddServicePayload } from 'types/services.types';
import { toScrapedService } from '../ServiceForm.utils';
import type { HAProxyFormValues } from './HAProxyForm.types';

export const toHAProxyPayload = (
  values: HAProxyFormValues
): AddServicePayload => ({
  haproxy: {
    ...toScrapedService(values, values.address.trim()),
    listenPort: Number(values.port),
    username: values.username.trim(),
    password: values.password.trim(),
  },
});
