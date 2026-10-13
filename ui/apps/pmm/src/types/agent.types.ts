export enum AgentUpdateSeverity {
  UNSPECIFIED = 'UPDATE_SEVERITY_UNSPECIFIED',
  UNSUPPORTED = 'UPDATE_SEVERITY_UNSUPPORTED',
  UP_TO_DATE = 'UPDATE_SEVERITY_UP_TO_DATE',
  REQUIRED = 'UPDATE_SEVERITY_REQUIRED',
  CRITICAL = 'UPDATE_SEVERITY_CRITICAL',
}

export interface GetAgentVersionItem {
  agentId: string;
  version: string;
  nodeName: string;
  severity: AgentUpdateSeverity;
}

export interface GetAgentVersionsResponse {
  agentVersions: GetAgentVersionItem[];
}

export enum AgentType {
  pmmAgent = 'pmm-agent',
  nodeExporter = 'node_exporter',
  mysqldExporter = 'mysqld_exporter',
  mongodbExporter = 'mongodb_exporter',
  postgresExporter = 'postgres_exporter',
  proxysqlExporter = 'proxysql_exporter',
  rdsExporter = 'rds_exporter',
  azureDatabaseExporter = 'azure_database_exporter',
  qanMysqlPerfschemaAgent = 'qan-mysql-perfschema-agent',
  qanMysqlSlowlogAgent = 'qan-mysql-slowlog-agent',
  qanMongodbProfilerAgent = 'qan-mongodb-profiler-agent',
  qanMongodbMongologAgent = 'qan-mongodb-mongolog-agent',
  qanPostgresqlPgstatementsAgent = 'qan-postgresql-pgstatements-agent',
  qanPostgresqlPgstatmonitorAgent = 'qan-postgresql-pgstatmonitor-agent',
  externalExporter = 'external-exporter',
  vmAgent = 'vmagent',
  nomadAgent = 'nomad-agent',
  valkeyExporter = 'valkey_exporter',
  rtaMongodbAgent = 'rta-mongodb-agent',
}

export enum AgentStatus {
  starting = 'AGENT_STATUS_STARTING',
  initializationError = 'AGENT_STATUS_INITIALIZATION_ERROR',
  running = 'AGENT_STATUS_RUNNING',
  waiting = 'AGENT_STATUS_WAITING',
  stopping = 'AGENT_STATUS_STOPPING',
  done = 'AGENT_STATUS_DONE',
  unknown = 'AGENT_STATUS_UNKNOWN',
}

// Agent from the /v1/management APIs (agents, nodes and services lists)
export interface ManagedAgent {
  agentId: string;
  agentType: AgentType;
  status: AgentStatus;
  isConnected: boolean;
  disabled: boolean;
  // Custom labels plus the agent's other parameters, keyed as the API names them
  properties: Record<string, string>;
}

export interface ListAgentsParams {
  serviceId?: string;
  nodeId?: string;
}

export interface RemoveAgentParams {
  agentId: string;
  force: boolean;
}

// The rds_exporter of a remote RDS node; the API only says whether a secret key is set
export interface RdsExporter {
  agentId: string;
  awsAccessKey: string;
  awsRoleArn: string;
  isAwsSecretKeySet: boolean;
}

// ChangeAgent only applies the fields it is sent, and an empty string clears one
export interface RdsCredentials {
  awsAccessKey?: string;
  awsSecretKey?: string;
  awsRoleArn?: string;
}

export interface ChangeRdsCredentialsParams {
  agentId: string;
  credentials: RdsCredentials;
}
