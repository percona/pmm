import { RealtimeSession, RealtimeSessionStatus } from 'types/rta.types';
import { ServiceType } from 'types/services.types';
import { Messages } from './RealtimeOverview.messages';

interface Selection {
  serviceIds: string[];
  serviceType?: ServiceType;
}

// A URL can name services of both technologies; the first one with a running
// session decides which technology the view shows.
export const resolveSelection = (
  serviceIds: string[],
  sessions: RealtimeSession[]
): Selection => {
  if (serviceIds.length === 0 || sessions.length === 0) {
    return { serviceIds };
  }

  const typeByServiceId = new Map(
    sessions.map((session) => [session.serviceId, session.serviceType])
  );
  const serviceType = serviceIds
    .map((serviceId) => typeByServiceId.get(serviceId))
    .find(Boolean);

  if (!serviceType) {
    return { serviceIds };
  }

  return {
    serviceIds: serviceIds.filter(
      // A service with no running session is no evidence of a mixed selection.
      (serviceId) =>
        (typeByServiceId.get(serviceId) ?? serviceType) === serviceType
    ),
    serviceType,
  };
};

// The agent's reasons, so an empty view of failed sessions does not read as an
// idle server.
export const sessionErrorsMessage = (
  serviceIds: string[],
  sessions: RealtimeSession[]
): string | undefined => {
  const reasons = sessions
    .filter(
      (session) =>
        serviceIds.includes(session.serviceId) &&
        session.status === RealtimeSessionStatus.error &&
        !!session.statusMessage
    )
    .map((session) =>
      Messages.sessionError(session.serviceName, session.statusMessage ?? '')
    );

  return reasons.length > 0 ? reasons.join('\n') : undefined;
};

// An unreadable lock source needs a configuration change, an unattributed row only a refresh;
// the tooltip must not send the reader to fix a configuration that is fine.
export const blockedOnlyTooltip = (
  sourcesMissing: boolean,
  unattributedCount: number
): string => {
  if (sourcesMissing && unattributedCount > 0) {
    return `${Messages.blockedPartialTooltip} ${Messages.blockedUnattributedNote(unattributedCount)}`;
  }

  if (sourcesMissing) {
    return Messages.blockedPartialTooltip;
  }

  if (unattributedCount > 0) {
    return Messages.blockedUnattributedTooltip(unattributedCount);
  }

  return Messages.blockedOnlyTooltip;
};
