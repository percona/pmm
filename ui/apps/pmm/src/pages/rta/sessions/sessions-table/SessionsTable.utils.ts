import { RealTimeSession, RealTimeSessionStatus } from 'types/rta.types';
import { SessionRow } from './SessionsTable.types';

export const getSessionRows = (
  sessions: RealTimeSession[]
): SessionRow[] => {
  const clusters = getClusters(sessions);
  const rows: SessionRow[] = [];

  for (const cluster of Object.keys(clusters)) {
    // non clustered services
    if (cluster === '') {
      for (const session of clusters[cluster]) {
        rows.push(serviceToSessionRow(session));
      }
    } else {
      const services = clusters[cluster].map(serviceToSessionRow);
      const areAllRunning = services.every(
        (service) => service.status === RealTimeSessionStatus.running
      );
      const earliestStartedAt = services.reduce((acc, service) => {
        return acc < service.startTime ? acc : service.startTime;
      }, services[0].startTime);

      rows.push({
        sessionId: cluster,
        type: 'cluster',
        sessionName: cluster,
        status: areAllRunning ? RealTimeSessionStatus.running : RealTimeSessionStatus.unspecified,
        startTime: earliestStartedAt,
        serviceSessions: services,
      });
    }
  }

  return rows;
};

const getClusters = (
  sessions: RealTimeSession[]
): Record<string, RealTimeSession[]> =>
  sessions.reduce<Record<string, RealTimeSession[]>>((acc, session) => {
    const key = session.clusterName || '';

    if (!acc[key]) {
      acc[key] = [];
    }

    acc[key].push(session);

    return acc;
  }, {});

const serviceToSessionRow = (serviceSession: RealTimeSession): SessionRow => ({
  sessionId: serviceSession.serviceId,
  type: 'service',
  sessionName: serviceSession.serviceName,
  status: serviceSession.status,
  startTime: serviceSession.startTime,
  serviceSessions: [],
});


export const getServiceIds = (session: SessionRow | SessionRow[]): string[] => {
  if (Array.isArray(session)) {
    return session.flatMap((session) => getServiceIds(session));
  }

  return session.type === 'service' ? [session.sessionId] : session.serviceSessions.map((session) => session.sessionId);
};

export const getAllSessions = (rows: SessionRow[]): SessionRow[] => rows.flatMap((session) =>
  session.type === 'service' ? [session] : session.serviceSessions
);