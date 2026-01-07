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
      sessions.push({
        sessionId: cluster,
        type: 'cluster',
        sessionName: cluster,
        // todo: get status from cluster
        status: AgentStatus.RUNNING,
        serviceSessions: clusters[cluster].map(agentToSession),
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
});
