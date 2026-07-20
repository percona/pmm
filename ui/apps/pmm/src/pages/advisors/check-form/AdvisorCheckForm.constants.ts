import { AdvisorFamily, AdvisorInterval } from 'types/advisors.types';

export const FAMILY_OPTIONS: AdvisorFamily[] = [
  AdvisorFamily.mysql,
  AdvisorFamily.postgresql,
  AdvisorFamily.mongodb,
];

export const INTERVAL_OPTIONS: AdvisorInterval[] = [
  AdvisorInterval.standard,
  AdvisorInterval.rare,
  AdvisorInterval.frequent,
];

// query types available regardless of family
const SHARED_QUERY_TYPES = [
  'METRICS_INSTANT',
  'METRICS_RANGE',
  'CLICKHOUSE_SELECT',
];

// query types offered per family; values match the server-side check.Type constants
export const QUERY_TYPES_BY_FAMILY: Record<AdvisorFamily, string[]> = {
  [AdvisorFamily.mysql]: ['MYSQL_SHOW', 'MYSQL_SELECT', ...SHARED_QUERY_TYPES],
  [AdvisorFamily.postgresql]: [
    'POSTGRESQL_SHOW',
    'POSTGRESQL_SELECT',
    ...SHARED_QUERY_TYPES,
  ],
  [AdvisorFamily.mongodb]: [
    'MONGODB_GETPARAMETER',
    'MONGODB_BUILDINFO',
    'MONGODB_GETCMDLINEOPTS',
    'MONGODB_GETDIAGNOSTICDATA',
    'MONGODB_REPLSETGETSTATUS',
    ...SHARED_QUERY_TYPES,
  ],
  [AdvisorFamily.unspecified]: [],
};
