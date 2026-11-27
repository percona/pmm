import type {
  AddServicePayload,
  DiscoverRDSInstance,
  DiscoverRDSPayload,
} from 'types/services.types';
import {
  POSTGRESQL_OPTIONS_DEFAULTS,
  POSTGRESQL_PORT,
} from '../../postgresql/PostgreSQLForm.constants';
import { TrackingOption } from '../../postgresql/PostgreSQLForm.types';
import {
  toAutoDiscoveryLimit,
  toMaxExporterConnections,
} from '../../postgresql/PostgreSQLForm.utils';
import { DATABASE_SERVICE_DEFAULTS } from '../../ServiceForm.constants';
import { toCollectors, toOptionalString } from '../../ServiceForm.utils';
import { toRdsDefaults, toRdsService } from '../RdsServiceForm.utils';
import type { RdsPostgreSQLFormValues } from './RdsPostgreSQLForm.types';

export const toRdsPostgreSQLDefaults = (
  instance: DiscoverRDSInstance
): RdsPostgreSQLFormValues => ({
  ...DATABASE_SERVICE_DEFAULTS,
  ...POSTGRESQL_OPTIONS_DEFAULTS,
  tracking: TrackingOption.pgStatements,
  ...toRdsDefaults(instance, POSTGRESQL_PORT),
});

export const toRdsPostgreSQLPayload = (
  values: RdsPostgreSQLFormValues,
  instance: DiscoverRDSInstance,
  credentials: DiscoverRDSPayload
): AddServicePayload => ({
  rds: {
    ...toRdsService(values, instance, credentials),
    database: toOptionalString(values.database),
    qanPostgresqlPgstatements: values.tracking === TrackingOption.pgStatements,
    autoDiscoveryLimit: toAutoDiscoveryLimit(values),
    maxPostgresqlExporterConnections: toMaxExporterConnections(values),
    postgresqlDisableCollectors: toCollectors(values.disableCollectors),
  },
});
