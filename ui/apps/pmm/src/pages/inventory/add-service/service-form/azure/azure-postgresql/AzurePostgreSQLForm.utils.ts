import {
  type AddAzureDatabasePayload,
  type DiscoverAzureInstance,
  type DiscoverAzurePayload,
  DiscoverAzureDatabaseType,
} from 'types/services.types';
import {
  POSTGRESQL_OPTIONS_DEFAULTS,
  POSTGRESQL_PORT,
} from '../../postgresql/PostgreSQLForm.constants';
import { TrackingOption } from '../../postgresql/PostgreSQLForm.types';
import { DATABASE_SERVICE_DEFAULTS } from '../../ServiceForm.constants';
import { toAzureDefaults, toAzureService } from '../AzureServiceForm.utils';
import type { AzurePostgreSQLFormValues } from './AzurePostgreSQLForm.types';

export const toAzurePostgreSQLDefaults = (
  instance: DiscoverAzureInstance
): AzurePostgreSQLFormValues => ({
  ...DATABASE_SERVICE_DEFAULTS,
  ...POSTGRESQL_OPTIONS_DEFAULTS,
  tracking: TrackingOption.pgStatements,
  ...toAzureDefaults(instance, POSTGRESQL_PORT),
});

export const toAzurePostgreSQLPayload = (
  values: AzurePostgreSQLFormValues,
  instance: DiscoverAzureInstance,
  credentials: DiscoverAzurePayload
): AddAzureDatabasePayload => ({
  ...toAzureService(values, instance, credentials),
  type: DiscoverAzureDatabaseType.postgresql,
  qan: values.tracking === TrackingOption.pgStatements,
});
