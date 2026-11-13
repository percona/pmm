import type {
  AddServicePayload,
  DiscoverRDSInstance,
  DiscoverRDSPayload,
} from 'types/services.types';
import {
  MYSQL_OPTIONS_DEFAULTS,
  MYSQL_PORT,
} from '../../mysql/MySQLForm.constants';
import { toTablestatsLimit } from '../../mysql/MySQLForm.utils';
import { DATABASE_SERVICE_DEFAULTS } from '../../ServiceForm.constants';
import { toCollectors } from '../../ServiceForm.utils';
import { toRdsDefaults, toRdsService } from '../RdsServiceForm.utils';
import type { RdsMySQLFormValues } from './RdsMySQLForm.types';

export const toRdsMySQLDefaults = (
  instance: DiscoverRDSInstance
): RdsMySQLFormValues => ({
  ...DATABASE_SERVICE_DEFAULTS,
  ...MYSQL_OPTIONS_DEFAULTS,
  ...toRdsDefaults(instance, MYSQL_PORT),
});

export const toRdsMySQLPayload = (
  values: RdsMySQLFormValues,
  instance: DiscoverRDSInstance,
  credentials: DiscoverRDSPayload
): AddServicePayload => ({
  rds: {
    ...toRdsService(values, instance, credentials),
    qanMysqlPerfschema: values.qanMysqlPerfschema,
    tablestatsGroupTableLimit: toTablestatsLimit(values),
    mysqlDisableCollectors: toCollectors(values.disableCollectors),
  },
});
