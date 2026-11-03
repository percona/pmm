import { z } from 'zod';
import {
  databaseServiceSchema,
  disableCollectorsSchema,
  keyValuesText,
  maxQueryLengthSchema,
  optionalIntegerText,
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
    tablestatsGroupTableLimit: optionalIntegerText(-1),
  })
  .superRefine(requireCredentialsWithoutTls)
  .superRefine(
    requireWhen(
      (values) => values.tablestatOptions === TablestatOption.custom,
      'tablestatsGroupTableLimit'
    )
  );
