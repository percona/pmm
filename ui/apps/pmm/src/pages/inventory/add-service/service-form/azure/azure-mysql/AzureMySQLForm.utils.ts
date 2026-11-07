import {
  type AddAzureDatabasePayload,
  type DiscoverAzureInstance,
  type DiscoverAzurePayload,
  DiscoverAzureDatabaseType,
} from 'types/services.types';
import {
  MYSQL_OPTIONS_DEFAULTS,
  MYSQL_PORT,
} from '../../mysql/MySQLForm.constants';
import { toTablestatsLimit } from '../../mysql/MySQLForm.utils';
import { DATABASE_SERVICE_DEFAULTS } from '../../ServiceForm.constants';
import { toAzureDefaults, toAzureService } from '../AzureServiceForm.utils';
import type { AzureMySQLFormValues } from './AzureMySQLForm.types';

export const toAzureMySQLDefaults = (
  instance: DiscoverAzureInstance
): AzureMySQLFormValues => ({
  ...DATABASE_SERVICE_DEFAULTS,
  ...MYSQL_OPTIONS_DEFAULTS,
  ...toAzureDefaults(instance, MYSQL_PORT),
});

export const toAzureMySQLPayload = (
  values: AzureMySQLFormValues,
  instance: DiscoverAzureInstance,
  credentials: DiscoverAzurePayload
): AddAzureDatabasePayload => ({
  ...toAzureService(values, instance, credentials),
  type: DiscoverAzureDatabaseType.mysql,
  qan: values.qanMysqlPerfschema,
  tablestatsGroupTableLimit: toTablestatsLimit(values),
});
