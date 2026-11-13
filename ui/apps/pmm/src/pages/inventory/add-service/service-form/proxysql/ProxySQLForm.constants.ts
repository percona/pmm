import {
  DATABASE_SERVICE_DEFAULTS,
  DISABLE_COLLECTORS_DEFAULTS,
} from '../ServiceForm.constants';
import type { ProxySQLFormValues } from './ProxySQLForm.types';

export const PROXYSQL_DEFAULTS: ProxySQLFormValues = {
  ...DATABASE_SERVICE_DEFAULTS,
  port: '6032',
  ...DISABLE_COLLECTORS_DEFAULTS,
};
