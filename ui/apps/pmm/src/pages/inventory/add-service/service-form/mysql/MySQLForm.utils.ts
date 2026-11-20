import type { AddServicePayload } from 'types/services.types';
import {
  toCollectors,
  toDatabaseService,
  toKeyValueMap,
  toLimit,
  toOptionalNumber,
  toTlsCertificates,
} from '../ServiceForm.utils';
import { TABLESTAT_LIMITS } from './MySQLForm.constants';
import {
  type MySQLFormValues,
  type MySQLOptionsFormValues,
  TablestatOption,
} from './MySQLForm.types';

export const toTablestatsLimit = (values: MySQLOptionsFormValues) =>
  toLimit(
    values.tablestatOptions,
    TablestatOption.custom,
    values.tablestatsGroupTableLimit,
    TABLESTAT_LIMITS
  );

export const toMySQLPayload = (values: MySQLFormValues): AddServicePayload => ({
  mysql: {
    ...toDatabaseService(values),
    ...toTlsCertificates(values),
    qanMysqlPerfschema: values.qanMysqlPerfschema,
    disableQueryExamples: values.disableQueryExamples,
    disableCommentsParsing: values.disableCommentsParsing,
    maxQueryLength: toOptionalNumber(values.maxQueryLength),
    extraDsnParams: toKeyValueMap(values.extraDsnParams),
    disableCollectors: toCollectors(values.disableCollectors),
    tablestatsGroupTableLimit: toTablestatsLimit(values),
  },
});
