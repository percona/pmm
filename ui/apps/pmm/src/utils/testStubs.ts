import {
  QueryData,
  RealtimeSession,
  RealtimeSessionStatus,
} from 'types/rta.types';
import {
  BaseService,
  ListServicesResponse,
  ManagedService,
  ManagedServicesResponse,
  ManagedServiceType,
  MySqlService,
} from 'types/services.types';
import { OrgRole, User } from 'types/user.types';

export const TEST_USER_ADMIN: User = {
  id: 1,
  login: 'admin',
  name: 'admin',
  isAuthorized: true,
  isViewer: true,
  isEditor: true,
  isPMMAdmin: true,
  orgId: 1,
  orgRole: OrgRole.Admin,
  orgs: [],
  preferences: {},
  info: {
    userId: 0,
    productTourCompleted: false,
    alertingTourCompleted: false,
    snoozedAt: null,
    snoozeCount: 0,
    snoozedPmmVersion: '',
  },
};

export const TEST_USER_EDITOR: User = {
  ...TEST_USER_ADMIN,
  id: 2,
  login: 'editor',
  name: 'editor',
  isPMMAdmin: false,
  orgId: 1,
  orgRole: OrgRole.Editor,
  info: {
    ...TEST_USER_ADMIN.info,
    userId: 2,
  },
};

export const TEST_USER_VIEWER: User = {
  ...TEST_USER_ADMIN,
  id: 3,
  login: 'viewer',
  name: 'viewer',
  isEditor: false,
  isPMMAdmin: false,
  orgRole: OrgRole.Viewer,
  info: {
    ...TEST_USER_ADMIN.info,
    userId: 3,
  },
};

export const TEST_SERVICE: BaseService = {
  serviceId: 'service-1',
  serviceName: 'Service 1',
  nodeId: 'node-1',
  environment: 'production',
  cluster: 'cluster-1',
  replicationSet: 'replication-set-1',
  customLabels: {},
};

// Managed services response format (from /v1/management/services API)
export const TEST_MANAGED_SERVICES: ManagedServicesResponse = {
  services: [],
};

export const TEST_MANAGED_SERVICE: ManagedService = {
  serviceId: 'service-1',
  serviceType: 'mysql',
  serviceName: 'Service 1',
  databaseName: '',
  nodeId: 'node-1',
  nodeName: 'Node 1',
  environment: 'production',
  cluster: 'cluster-1',
  replicationSet: 'replication-set-1',
  customLabels: {},
  externalGroup: '',
  address: '127.0.0.1',
  port: 3306,
  socket: '',
  version: '8.0.0',
};

export const TEST_MANAGED_SERVICE_MONGO: ManagedService = {
  ...TEST_MANAGED_SERVICE,
  serviceType: ManagedServiceType.mongodb,
};

export const TEST_MANAGED_SERVICES_WITH_ONE_MYSQL: ManagedServicesResponse = {
  services: [TEST_MANAGED_SERVICE],
};

// Inventory services response format (from /v1/inventory/services API)
export const TEST_SERVICES: ListServicesResponse = {
  mysql: [],
  mongodb: [],
  postgresql: [],
  proxysql: [],
  haproxy: [],
  external: [],
  valkey: [],
};

export const TEST_SERVICES_WITH_ONE_MYSQL: ListServicesResponse = {
  ...TEST_SERVICES,
  mysql: [TEST_SERVICE as MySqlService],
};

export const TEST_REAL_TIME_SESSION: RealtimeSession = {
  serviceId: 'service-1',
  serviceName: 'Service 1',
  clusterName: 'cluster-1',
  startTime: '2021-01-01T00:00:00Z',
  status: RealtimeSessionStatus.unspecified,
};

export const TEST_REAL_TIME_SESSION_2: RealtimeSession = {
  serviceId: 'service-2',
  serviceName: 'Service 2',
  clusterName: 'cluster-2',
  startTime: '2021-01-01T00:00:00Z',
  status: RealtimeSessionStatus.unspecified,
};

export const TEST_MONGO_DB_QUERY_DATA: QueryData = {
  serviceId: 'service-1',
  serviceName: 'Service 1',
  queryId: 'query-1',
  queryText: '{ find: "mycollection", filter: { status: "active" } }',
  state: 'running',
  executionDuration: '10s',
  rowsExamined: 100,
  rowsSent: 100,
  collectTime: '2021-01-01T00:00:00Z',
  rawQueryJson: '{ find: "mycollection", filter: { status: "active" } }',
  mongoDbPayload: {
    opid: '1',
    client: '127.0.0.1',
    waitingForLock: false,
    indexUtilized: 'COLLSCAN',
  },
};
