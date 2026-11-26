import { AgentType } from 'types/agent.types';

export const Messages = {
  goBackToServices: 'Go back to services',
  goBackToNodes: 'Go back to nodes',
  serviceTitle: (serviceName: string) => `Service ${serviceName} / Agents`,
  nodeTitle: (nodeName: string) => `Node ${nodeName} / Agents`,
  columns: {
    status: 'Status',
    agentType: 'Agent Type',
    agentId: 'Agent ID',
  },
  details: {
    properties: 'Properties',
  },
  agentTypes: {
    [AgentType.pmmAgent]: 'PMM agent',
    [AgentType.nodeExporter]: 'Node exporter',
    [AgentType.mysqldExporter]: 'MySQL exporter',
    [AgentType.mongodbExporter]: 'MongoDB exporter',
    [AgentType.postgresExporter]: 'PostgreSQL exporter',
    [AgentType.proxysqlExporter]: 'ProxySQL exporter',
    [AgentType.rdsExporter]: 'RDS exporter',
    [AgentType.azureDatabaseExporter]: 'Azure database exporter',
    [AgentType.qanMysqlPerfschemaAgent]: 'QAN MySQL Performance Schema agent',
    [AgentType.qanMysqlSlowlogAgent]: 'QAN MySQL Slow Query Log agent',
    [AgentType.qanMongodbProfilerAgent]: 'QAN MongoDB Profiler agent',
    [AgentType.qanMongodbMongologAgent]: 'QAN MongoDB Mongo Log agent',
    [AgentType.qanPostgresqlPgstatementsAgent]:
      'QAN PostgreSQL pg_stat_statements agent',
    [AgentType.qanPostgresqlPgstatmonitorAgent]:
      'QAN PostgreSQL pg_stat_monitor agent',
    [AgentType.externalExporter]: 'External exporter',
    [AgentType.vmAgent]: 'VictoriaMetrics agent',
    [AgentType.nomadAgent]: 'Nomad agent',
    [AgentType.valkeyExporter]: 'Valkey exporter',
    [AgentType.rtaMongodbAgent]: 'Real-Time Analytics MongoDB agent',
  } satisfies Record<AgentType, string>,
  empty: 'No agents available',
  delete: {
    title: 'Confirm action',
    message: (count: number) =>
      `Are you sure that you want to permanently delete ${count} agent${count > 1 ? 's' : ''}?`,
    forceDescription: 'Force mode is going to delete all associated agents',
    confirm: 'Proceed',
    success: (deleted: number, total: number) =>
      `${deleted} of ${total} agents successfully deleted`,
  },
};
