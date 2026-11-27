import {
  COMMON_OPTIONS_DEFAULTS,
  LABELS_DEFAULTS,
} from '../ServiceForm.constants';
import type { HAProxyFormValues } from './HAProxyForm.types';

export const HAPROXY_DEFAULTS: HAProxyFormValues = {
  serviceName: '',
  address: '',
  port: '8404',
  username: '',
  password: '',
  ...LABELS_DEFAULTS,
  ...COMMON_OPTIONS_DEFAULTS,
};
