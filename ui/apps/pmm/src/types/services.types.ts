export enum ServiceType {
  unspecified = 'SERVICE_TYPE_UNSPECIFIED',
  mysql = 'SERVICE_TYPE_MYSQL_SERVICE',
  mongodb = 'SERVICE_TYPE_MONGODB_SERVICE',
  posgresql = 'SERVICE_TYPE_POSTGRESQL_SERVICE',
  proxysql = 'SERVICE_TYPE_PROXYSQL_SERVICE',
  haproxy = 'SERVICE_TYPE_HAPROXY_SERVICE',
  external = 'SERVICE_TYPE_EXTERNAL_SERVICE',
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

export interface ListServicesResponse {
  mysql: MySqlService[];
  mongodb: VersionedService[];
  postgresql: PostgreSqlService[];
  proxysql: ProxySqlService[];
  haproxy: HaProxyService[];
  external: ExternalService[];
  valkey: ValkeyService[];
}

export interface ListServicesParams {
  nodeId?: string;
  externalGroup?: string;
  serviceType?: ServiceType;
}
