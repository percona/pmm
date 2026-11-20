import {
  DATABASE_SERVICE_DEFAULTS,
  TLS_CERTIFICATES_DEFAULTS,
} from '../ServiceForm.constants';
import type { ValkeyFormValues } from './ValkeyForm.types';

export const VALKEY_DEFAULTS: ValkeyFormValues = {
  ...DATABASE_SERVICE_DEFAULTS,
  port: '6379',
  ...TLS_CERTIFICATES_DEFAULTS,
};
