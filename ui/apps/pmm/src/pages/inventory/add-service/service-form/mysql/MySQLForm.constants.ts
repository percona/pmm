import {
  DATABASE_SERVICE_DEFAULTS,
  DISABLE_COLLECTORS_DEFAULTS,
  MAX_QUERY_LENGTH_DEFAULTS,
  QUERY_ANALYTICS_OPTIONS_DEFAULTS,
  TLS_CERTIFICATES_DEFAULTS,
} from '../ServiceForm.constants';
import { toRadioOptions } from '../ServiceForm.utils';
import { Messages } from './MySQLForm.messages';
import {
  type MySQLFormValues,
  type MySQLOptionsFormValues,
  TablestatOption,
} from './MySQLForm.types';

export const MYSQL_PORT = '3306';

export const TABLESTAT_OPTIONS = toRadioOptions(
  Object.values(TablestatOption),
  Messages.tablestats.options
);

export const TABLESTAT_LIMITS = {
  [TablestatOption.disabled]: -1,
  [TablestatOption.default]: 1000,
};

export const MYSQL_OPTIONS_DEFAULTS: MySQLOptionsFormValues = {
  ...TLS_CERTIFICATES_DEFAULTS,
  ...QUERY_ANALYTICS_OPTIONS_DEFAULTS,
  disableCommentsParsing: true,
  ...DISABLE_COLLECTORS_DEFAULTS,
  ...MAX_QUERY_LENGTH_DEFAULTS,
  qanMysqlPerfschema: true,
  extraDsnParams: '',
  tablestatOptions: TablestatOption.disabled,
  tablestatsGroupTableLimit: '1000',
};

export const MYSQL_DEFAULTS: MySQLFormValues = {
  ...DATABASE_SERVICE_DEFAULTS,
  port: MYSQL_PORT,
  ...MYSQL_OPTIONS_DEFAULTS,
};
