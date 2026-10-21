import { AgentType, LogLevel } from 'types/agent.types';
import { ManagedServiceType } from 'types/services.types';
import {
  QuerySource,
  type QuerySourceOption,
  type SettingName,
} from './EditService.types';

const LABEL_SETTINGS: SettingName[] = [
  'environment',
  'cluster',
  'replicationSet',
  'customLabels',
];

const DATABASE_SETTINGS: SettingName[] = [
  ...LABEL_SETTINGS,
  'username',
  'password',
  'agentPassword',
  'tls',
  'tlsSkipVerify',
  'metricsMode',
  'logLevel',
  'exposeExporter',
  'connectionTimeout',
];

const ENDPOINT_SETTINGS: SettingName[] = [
  ...LABEL_SETTINGS,
  'username',
  'password',
  'tlsSkipVerify',
  'metricsMode',
  'scheme',
  'metricsPath',
  'listenPort',
];

// The settings `pmm-admin update` changes for each service type, except the
// address, port and socket, and the exporter's environment variables
export const SERVICE_SETTINGS: Record<ManagedServiceType, SettingName[]> = {
  [ManagedServiceType.mysql]: [
    ...DATABASE_SETTINGS,
    'extraDsnParams',
    'tlsCa',
    'tlsCert',
    'tlsKey',
    'querySource',
    'maxQueryLength',
    'disableQueryExamples',
    'disableCommentsParsing',
    'maxSlowlogFileSize',
    'disableCollectors',
    'tablestatsGroupTableLimit',
  ],
  [ManagedServiceType.mongodb]: [
    ...DATABASE_SETTINGS,
    'authenticationMechanism',
    'authenticationDatabase',
    'tlsCa',
    'tlsCertificateKey',
    'tlsCertificateKeyFilePassword',
    'querySource',
    'maxQueryLength',
    'disableCollectors',
    'enableAllCollectors',
    'enableDiagnosticDataHistograms',
    'statsCollections',
    'collectionsLimit',
  ],
  [ManagedServiceType.postgresql]: [
    ...DATABASE_SETTINGS,
    'database',
    'tlsCa',
    'tlsCert',
    'tlsKey',
    'querySource',
    'maxQueryLength',
    'disableQueryExamples',
    'disableCommentsParsing',
    'disableCollectors',
    'autoDiscoveryLimit',
    'maxExporterConnections',
  ],
  [ManagedServiceType.proxysql]: [...DATABASE_SETTINGS, 'disableCollectors'],
  [ManagedServiceType.valkey]: [
    ...DATABASE_SETTINGS,
    'tlsCa',
    'tlsCert',
    'tlsKey',
  ],
  [ManagedServiceType.haproxy]: ENDPOINT_SETTINGS,
  [ManagedServiceType.external]: [...ENDPOINT_SETTINGS, 'group'],
};

export const EXPORTER_TYPES: Record<ManagedServiceType, AgentType> = {
  [ManagedServiceType.mysql]: AgentType.mysqldExporter,
  [ManagedServiceType.mongodb]: AgentType.mongodbExporter,
  [ManagedServiceType.postgresql]: AgentType.postgresExporter,
  [ManagedServiceType.proxysql]: AgentType.proxysqlExporter,
  [ManagedServiceType.valkey]: AgentType.valkeyExporter,
  [ManagedServiceType.haproxy]: AgentType.externalExporter,
  [ManagedServiceType.external]: AgentType.externalExporter,
};

export const QUERY_SOURCES: Partial<
  Record<ManagedServiceType, QuerySourceOption[]>
> = {
  [ManagedServiceType.mysql]: [
    {
      value: QuerySource.perfschema,
      agentType: AgentType.qanMysqlPerfschemaAgent,
      change: 'qanMysqlPerfschema',
      isCloud: true,
    },
    {
      value: QuerySource.slowlog,
      agentType: AgentType.qanMysqlSlowlogAgent,
      change: 'qanMysqlSlowlog',
    },
  ],
  [ManagedServiceType.mongodb]: [
    {
      value: QuerySource.profiler,
      agentType: AgentType.qanMongodbProfilerAgent,
      change: 'qanMongodbProfiler',
    },
    {
      value: QuerySource.mongolog,
      agentType: AgentType.qanMongodbMongologAgent,
      change: 'qanMongodbMongolog',
    },
  ],
  [ManagedServiceType.postgresql]: [
    {
      value: QuerySource.pgstatements,
      agentType: AgentType.qanPostgresqlPgstatementsAgent,
      change: 'qanPostgresqlPgstatementsAgent',
      isCloud: true,
    },
    {
      value: QuerySource.pgstatmonitor,
      agentType: AgentType.qanPostgresqlPgstatmonitorAgent,
      change: 'qanPostgresqlPgstatmonitorAgent',
    },
  ],
};

// The exporters that only log from the error level up; the server raises a
// lower level to it
const NO_FATAL_TYPES = [
  ManagedServiceType.mysql,
  ManagedServiceType.postgresql,
  ManagedServiceType.valkey,
];

export const getLogLevels = (serviceType: ManagedServiceType) => [
  LogLevel.unspecified,
  ...(NO_FATAL_TYPES.includes(serviceType) ? [] : [LogLevel.fatal]),
  LogLevel.error,
  LogLevel.warn,
  LogLevel.info,
  LogLevel.debug,
];

// Its query source is pinned by PMM_ENABLE_INTERNAL_PG_QAN and Settings
export const PMM_SERVER_POSTGRESQL = 'pmm-server-postgresql';

// Whole-number settings, with the lowest value the API takes for each
export const NUMBER_SETTINGS: Partial<Record<SettingName, number>> = {
  listenPort: 1,
  maxQueryLength: -1,
  tablestatsGroupTableLimit: -1,
  collectionsLimit: -1,
  autoDiscoveryLimit: -1,
  maxExporterConnections: 0,
};

export const MAX_PORT = 65535;

export const MIB = 1024 * 1024;

// What the server stores for a new slowlog QAN agent
export const DEFAULT_SLOWLOG_FILE_SIZE = 1024 * MIB;

export const METRICS_SCHEMES = ['http', 'https'];
