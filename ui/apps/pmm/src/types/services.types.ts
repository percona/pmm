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
