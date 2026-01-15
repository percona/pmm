import { AgentStatus } from 'types/agent.types';
import { RealTimeSession, RunningRealTimeAgent } from 'types/rta.types';

export const getSessions = (
  agents: RunningRealTimeAgent[]
): RealTimeSession[] => {
  const clusters = getClusters(agents);
  const sessions: RealTimeSession[] = [];

  for (const cluster of Object.keys(clusters)) {
    // non clustered services
    if (cluster === '') {
      for (const agent of clusters[cluster]) {
        sessions.push(agentToSession(agent));
      }
    } else {
      const services = clusters[cluster].map(agentToSession);
      const areAllRunning = services.every(
        (service) => service.status === AgentStatus.RUNNING
      );
      const earliestStartedAt = services.reduce((acc, service) => {
        return acc < service.startedAt ? acc : service.startedAt;
      }, services[0].startedAt);

      sessions.push({
        sessionId: cluster,
        type: 'cluster',
        sessionName: cluster,
        status: areAllRunning ? AgentStatus.RUNNING : AgentStatus.UNKNOWN,
        serviceSessions: services,
        agents: clusters[cluster],
        startedAt: earliestStartedAt,
      });
    }
  }

  return sessions;
};

const getClusters = (
  agents: RunningRealTimeAgent[]
): Record<string, RunningRealTimeAgent[]> =>
  agents.reduce<Record<string, RunningRealTimeAgent[]>>((acc, agent) => {
    const key = agent.cluster || '';

    if (!acc[key]) {
      acc[key] = [];
    }

    acc[key].push(agent);

    return acc;
  }, {});

const agentToSession = (agent: RunningRealTimeAgent): RealTimeSession => ({
  sessionId: agent.agentId,
  type: 'service',
  sessionName: agent.serviceName,
  status: agent.status,
  serviceSessions: [],
  agents: [agent],
  startedAt: agent.startedAt,
});
