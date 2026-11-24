import type { LogLevel, ManagedAgent, ServiceAgent } from './agent.types';

export enum ServiceType {
  unspecified = 'SERVICE_TYPE_UNSPECIFIED',
  mysql = 'SERVICE_TYPE_MYSQL_SERVICE',
  mongodb = 'SERVICE_TYPE_MONGODB_SERVICE',
  posgresql = 'SERVICE_TYPE_POSTGRESQL_SERVICE',
  proxysql = 'SERVICE_TYPE_PROXYSQL_SERVICE',
  haproxy = 'SERVICE_TYPE_HAPROXY_SERVICE',
  valkey = 'SERVICE_TYPE_VALKEY_SERVICE',
  external = 'SERVICE_TYPE_EXTERNAL_SERVICE',
}

// Service types returned by /v1/management/services API (lowercase format)
export enum ManagedServiceType {
  mysql = 'mysql',
  mongodb = 'mongodb',
  postgresql = 'postgresql',
  proxysql = 'proxysql',
  haproxy = 'haproxy',
  valkey = 'valkey',
  external = 'external',
}

export enum ServiceStatus {
  unspecified = 'STATUS_UNSPECIFIED',
  up = 'STATUS_UP',
  down = 'STATUS_DOWN',
  unknown = 'STATUS_UNKNOWN',
}

export interface ListTypesResponse {
  serviceTypes: ServiceType[];
}

export interface BaseService {
  serviceId: string;
  serviceName: string;
  nodeId: string;
  environment: string;
  cluster: string;
  replicationSet: string;
  customLabels: Record<string, string>;
}

export interface NetworkService extends BaseService {
  address: string;
  port: number;
  socket: string;
}

export interface VersionedService extends NetworkService {
  version: string;
}

export interface MySqlService extends VersionedService {
  extraDsnParams: Record<string, string>;
}

export interface PostgreSqlService extends VersionedService {
  databaseName: string;
  autoDiscoveryLimit: number;
}

export interface ProxySqlService extends VersionedService {}

export interface HaProxyService extends BaseService {}

export interface ExternalService extends BaseService {
  group: string;
}

export interface ValkeyService extends VersionedService {}

// Service from /v1/management/services API
export interface ManagedService {
  serviceId: string;
  serviceType: ManagedServiceType;
  serviceName: string;
  databaseName: string;
  nodeId: string;
  nodeName: string;
  environment: string;
  cluster: string;
  replicationSet: string;
  customLabels: Record<string, string>;
  externalGroup: string;
  address: string;
  port: number;
  socket: string;
  version: string;
  status: ServiceStatus;
  agents: ManagedAgent[];
  // Custom labels plus the service's other scalar attributes, keyed as the API names them
  labels: Record<string, string>;
}

// Response from /v1/management/services API
export interface ManagedServicesResponse {
  services: ManagedService[];
}

// Response from /v1/inventory/services API
export interface ListServicesResponse {
  mysql?: MySqlService[];
  mongodb?: VersionedService[];
  postgresql?: PostgreSqlService[];
  proxysql?: ProxySqlService[];
  haproxy?: HaProxyService[];
  external?: ExternalService[];
  valkey?: ValkeyService[];
}

export interface ListServicesParams {
  nodeId?: string;
  externalGroup?: string;
  serviceType?: ServiceType;
}

export interface RemoveServiceParams {
  serviceId: string;
  force: boolean;
}

export enum MetricsMode {
  pull = 'METRICS_MODE_PULL',
  push = 'METRICS_MODE_PUSH',
}

// A service of any type from /v1/inventory/services/{id}; the fields its type
// lacks are empty
export interface InventoryService extends BaseService {
  serviceType: ManagedServiceType;
  address: string;
  port: number;
  socket: string;
  databaseName: string;
  group: string;
}

// A service with the settings of its agents
export interface EditableService {
  service: InventoryService;
  agents: ServiceAgent[];
}

// The settings UpdateService changes, named as the API names them. It only
// applies the fields it is sent; an empty string, map or list clears one.
export interface ServiceChanges {
  environment?: string;
  cluster?: string;
  replicationSet?: string;
  customLabels?: Record<string, string>;
  group?: string;
  database?: string;
  username?: string;
  password?: string;
  agentPassword?: string;
  authenticationMechanism?: string;
  authenticationDatabase?: string;
  extraDsnParams?: Record<string, string>;
  scheme?: string;
  metricsPath?: string;
  listenPort?: number;
  tls?: boolean;
  tlsSkipVerify?: boolean;
  tlsCa?: string;
  tlsCert?: string;
  tlsKey?: string;
  tlsCertificateKey?: string;
  tlsCertificateKeyFilePassword?: string;
  qanMysqlPerfschema?: boolean;
  qanMysqlSlowlog?: boolean;
  qanMongodbProfiler?: boolean;
  qanMongodbMongolog?: boolean;
  qanPostgresqlPgstatementsAgent?: boolean;
  qanPostgresqlPgstatmonitorAgent?: boolean;
  maxQueryLength?: number;
  disableQueryExamples?: boolean;
  disableCommentsParsing?: boolean;
  maxSlowlogFileSize?: number;
  metricsMode?: MetricsMode;
  logLevel?: LogLevel;
  exposeExporter?: boolean;
  connectionTimeout?: string;
  disableCollectors?: string[];
  tablestatsGroupTableLimit?: number;
  enableAllCollectors?: boolean;
  enableDiagnosticDataHistograms?: boolean;
  statsCollections?: string[];
  collectionsLimit?: number;
  autoDiscoveryLimit?: number;
  maxExporterConnections?: number;
  skipConnectionCheck?: boolean;
}

export interface UpdateServiceParams {
  serviceId: string;
  serviceType: ManagedServiceType;
  changes: ServiceChanges;
}
