import type { NodeType } from 'types/nodes.types';

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
  serviceType: string;
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
  status?: ServiceStatus;
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

export enum MetricsMode {
  unspecified = 'METRICS_MODE_UNSPECIFIED',
  pull = 'METRICS_MODE_PULL',
  push = 'METRICS_MODE_PUSH',
}

export enum DiscoverRDSEngine {
  unspecified = 'DISCOVER_RDS_ENGINE_UNSPECIFIED',
  mysql = 'DISCOVER_RDS_ENGINE_MYSQL',
  postgresql = 'DISCOVER_RDS_ENGINE_POSTGRESQL',
}

// MySQL also covers MariaDB servers.
export enum DiscoverAzureDatabaseType {
  unspecified = 'DISCOVER_AZURE_DATABASE_TYPE_UNSPECIFIED',
  mysql = 'DISCOVER_AZURE_DATABASE_TYPE_MYSQL',
  postgresql = 'DISCOVER_AZURE_DATABASE_TYPE_POSTGRESQL',
}

export interface AddNodeParams {
  nodeType: NodeType;
  nodeName: string;
  region?: string;
  az?: string;
  customLabels?: Record<string, string>;
}

interface ServiceLabels {
  environment?: string;
  cluster?: string;
  replicationSet?: string;
  customLabels?: Record<string, string>;
}

interface ServicePlacement {
  nodeId?: string;
  nodeName?: string;
  addNode?: AddNodeParams;
}

interface ServiceConnection extends ServiceLabels, ServicePlacement {
  serviceName: string;
  address: string;
  port: number;
  pmmAgentId: string;
  username?: string;
  password?: string;
  skipConnectionCheck?: boolean;
  tls?: boolean;
  tlsSkipVerify?: boolean;
  metricsMode?: MetricsMode;
  disableCollectors?: string[];
  connectionTimeout?: string;
}

export interface AddMySQLServiceParams extends ServiceConnection {
  qanMysqlPerfschema?: boolean;
  disableCommentsParsing?: boolean;
  disableQueryExamples?: boolean;
  maxQueryLength?: number;
  tlsCa?: string;
  tlsCert?: string;
  tlsKey?: string;
  tablestatsGroupTableLimit?: number;
  extraDsnParams?: Record<string, string>;
}

export interface AddPostgreSQLServiceParams extends ServiceConnection {
  database?: string;
  qanPostgresqlPgstatementsAgent?: boolean;
  qanPostgresqlPgstatmonitorAgent?: boolean;
  disableCommentsParsing?: boolean;
  disableQueryExamples?: boolean;
  maxQueryLength?: number;
  tlsCa?: string;
  tlsCert?: string;
  tlsKey?: string;
  autoDiscoveryLimit?: number;
  maxExporterConnections?: number;
}

export interface AddMongoDBServiceParams extends ServiceConnection {
  qanMongodbProfiler?: boolean;
  maxQueryLength?: number;
  tlsCertificateKey?: string;
  tlsCertificateKeyFilePassword?: string;
  tlsCa?: string;
  authenticationMechanism?: string;
}

export interface AddProxySQLServiceParams extends ServiceConnection {}

export interface AddValkeyServiceParams extends ServiceConnection {
  tlsCa?: string;
  tlsCert?: string;
  tlsKey?: string;
}

interface ScrapedServiceParams extends ServiceLabels, ServicePlacement {
  serviceName: string;
  address: string;
  listenPort: number;
  username?: string;
  password?: string;
  scheme?: string;
  metricsPath?: string;
  metricsMode?: MetricsMode;
  skipConnectionCheck?: boolean;
  tlsSkipVerify?: boolean;
}

export interface AddHAProxyServiceParams extends ScrapedServiceParams {}

export interface AddExternalServiceParams extends ScrapedServiceParams {
  group?: string;
}

export interface AddRDSServiceParams extends ServiceLabels {
  region: string;
  az?: string;
  instanceId: string;
  address: string;
  port: number;
  engine: DiscoverRDSEngine;
  pmmAgentId?: string;
  nodeName?: string;
  serviceName?: string;
  username: string;
  password?: string;
  awsAccessKey?: string;
  awsSecretKey?: string;
  rdsExporter?: boolean;
  qanMysqlPerfschema?: boolean;
  qanPostgresqlPgstatements?: boolean;
  skipConnectionCheck?: boolean;
  tls?: boolean;
  tlsSkipVerify?: boolean;
  disableQueryExamples?: boolean;
  disableCommentsParsing?: boolean;
  tablestatsGroupTableLimit?: number;
  disableBasicMetrics?: boolean;
  disableEnhancedMetrics?: boolean;
  metricsMode?: MetricsMode;
  database?: string;
  autoDiscoveryLimit?: number;
  maxPostgresqlExporterConnections?: number;
  connectionTimeout?: string;
  mysqlDisableCollectors?: string[];
  postgresqlDisableCollectors?: string[];
}

// The request body is a oneof: exactly one service key is set.
export type AddServicePayload =
  | { mysql: AddMySQLServiceParams }
  | { postgresql: AddPostgreSQLServiceParams }
  | { mongodb: AddMongoDBServiceParams }
  | { proxysql: AddProxySQLServiceParams }
  | { valkey: AddValkeyServiceParams }
  | { haproxy: AddHAProxyServiceParams }
  | { external: AddExternalServiceParams }
  | { rds: AddRDSServiceParams };

export interface AddAzureDatabasePayload {
  region: string;
  az?: string;
  instanceId: string;
  address: string;
  port: number;
  nodeName?: string;
  serviceName?: string;
  environment?: string;
  username: string;
  password?: string;
  azureClientId: string;
  azureClientSecret: string;
  azureTenantId: string;
  azureSubscriptionId: string;
  azureResourceGroup: string;
  azureDatabaseExporter?: boolean;
  qan?: boolean;
  customLabels?: Record<string, string>;
  skipConnectionCheck?: boolean;
  tls?: boolean;
  tlsSkipVerify?: boolean;
  disableQueryExamples?: boolean;
  tablestatsGroupTableLimit?: number;
  type: DiscoverAzureDatabaseType;
  connectionTimeout?: string;
  pmmAgentId?: string;
}

export interface DiscoverRDSPayload {
  awsAccessKey?: string;
  awsSecretKey?: string;
}

export interface DiscoverRDSInstance {
  region: string;
  az: string;
  instanceId: string;
  nodeModel: string;
  address: string;
  port: number;
  engine: DiscoverRDSEngine;
  engineVersion: string;
}

export interface DiscoverRDSResponse {
  rdsInstances: DiscoverRDSInstance[];
}

export interface DiscoverAzurePayload {
  azureClientId: string;
  azureClientSecret: string;
  azureTenantId: string;
  azureSubscriptionId: string;
}

export interface DiscoverAzureInstance {
  instanceId: string;
  region: string;
  serviceName: string;
  username: string;
  address: string;
  azureResourceGroup: string;
  environment: string;
  type: DiscoverAzureDatabaseType;
  az: string;
  nodeModel: string;
}

export interface DiscoverAzureResponse {
  azureDatabaseInstance: DiscoverAzureInstance[];
}
