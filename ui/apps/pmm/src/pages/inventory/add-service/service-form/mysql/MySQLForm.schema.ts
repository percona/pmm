import { z } from 'zod';
import {
  databaseServiceSchema,
  disableCollectorsSchema,
  getIntegerError,
  keyValuesText,
  maxQueryLengthSchema,
  queryAnalyticsOptionsSchema,
  requireCredentialsWithoutTls,
  requireWhen,
  tlsCertificatesSchema,
} from '../ServiceForm.schema';
import { TablestatOption } from './MySQLForm.types';

export const mysqlSchema = z
  .object({
    ...databaseServiceSchema.shape,
    ...tlsCertificatesSchema.shape,
    ...queryAnalyticsOptionsSchema.shape,
    ...disableCollectorsSchema.shape,
    ...maxQueryLengthSchema.shape,
    qanMysqlPerfschema: z.boolean(),
    extraDsnParams: keyValuesText,
    tablestatOptions: z.enum(TablestatOption),
    tablestatsGroupTableLimit: z.string(),
  })
  .superRefine(requireCredentialsWithoutTls)
  .superRefine(
    requireWhen(
      (values) => values.tablestatOptions === TablestatOption.custom,
      'tablestatsGroupTableLimit',
      (value) => getIntegerError(value, -1)
    )
  );
