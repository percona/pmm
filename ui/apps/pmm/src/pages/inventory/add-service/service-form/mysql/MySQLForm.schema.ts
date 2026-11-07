import { z } from 'zod';
import {
  databaseServiceSchema,
  disableCollectorsSchema,
  getIntegerError,
  keyValuesText,
  maxQueryLengthSchema,
  queryAnalyticsOptionsSchema,
  requireCredentials,
  requireWhen,
  tlsCertificatesSchema,
} from '../ServiceForm.schema';
import {
  type MySQLOptionsFormValues,
  TablestatOption,
} from './MySQLForm.types';

// The MySQL options, shared by the remote, RDS and Azure forms.
export const mysqlOptionsSchema = z.object({
  ...tlsCertificatesSchema.shape,
  ...queryAnalyticsOptionsSchema.shape,
  ...disableCollectorsSchema.shape,
  ...maxQueryLengthSchema.shape,
  qanMysqlPerfschema: z.boolean(),
  extraDsnParams: keyValuesText,
  tablestatOptions: z.enum(TablestatOption),
  tablestatsGroupTableLimit: z.string(),
});

export const requireTablestatsLimit = requireWhen<MySQLOptionsFormValues>(
  (values) => values.tablestatOptions === TablestatOption.custom,
  'tablestatsGroupTableLimit',
  (value) => getIntegerError(value, -1)
);

export const mysqlSchema = z
  .object({ ...databaseServiceSchema.shape, ...mysqlOptionsSchema.shape })
  .superRefine(requireCredentials)
  .superRefine(requireTablestatsLimit);
