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

// Agent types returned by /v1/management APIs (lowercase format)
export enum ManagedAgentType {
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
  vmagent = 'vmagent',
  nomadAgent = 'nomad-agent',
  valkeyExporter = 'valkey_exporter',
  rtaMongodbAgent = 'rta-mongodb-agent',
}

export enum AgentStatus {
  unspecified = 'AGENT_STATUS_UNSPECIFIED',
  starting = 'AGENT_STATUS_STARTING',
  initializationError = 'AGENT_STATUS_INITIALIZATION_ERROR',
  running = 'AGENT_STATUS_RUNNING',
  waiting = 'AGENT_STATUS_WAITING',
  stopping = 'AGENT_STATUS_STOPPING',
  done = 'AGENT_STATUS_DONE',
  unknown = 'AGENT_STATUS_UNKNOWN',
}
