import { AdvisorTechnology, AdvisorInterval } from 'types/advisors.types';

export const TECHNOLOGY_OPTIONS: AdvisorTechnology[] = [
  AdvisorTechnology.mysql,
  AdvisorTechnology.postgresql,
  AdvisorTechnology.mongodb,
];

export const INTERVAL_OPTIONS: AdvisorInterval[] = [
  AdvisorInterval.standard,
  AdvisorInterval.rare,
  AdvisorInterval.frequent,
];

// query types available regardless of technology
const SHARED_QUERY_TYPES = [
  'METRICS_INSTANT',
  'METRICS_RANGE',
  'CLICKHOUSE_SELECT',
];

// query types offered per technology; values match the server-side check.Type constants
export const QUERY_TYPES_BY_TECHNOLOGY: Record<AdvisorTechnology, string[]> = {
  [AdvisorTechnology.mysql]: [
    'MYSQL_SHOW',
    'MYSQL_SELECT',
    ...SHARED_QUERY_TYPES,
  ],
  [AdvisorTechnology.postgresql]: [
    'POSTGRESQL_SHOW',
    'POSTGRESQL_SELECT',
    ...SHARED_QUERY_TYPES,
  ],
  [AdvisorTechnology.mongodb]: [
    'MONGODB_GETPARAMETER',
    'MONGODB_BUILDINFO',
    'MONGODB_GETCMDLINEOPTS',
    'MONGODB_GETDIAGNOSTICDATA',
    'MONGODB_REPLSETGETSTATUS',
    ...SHARED_QUERY_TYPES,
  ],
  [AdvisorTechnology.unspecified]: [],
};
