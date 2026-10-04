import { RealtimeSession, RealtimeSessionStatus } from 'types/rta.types';
import { ServiceType } from 'types/services.types';
import { Messages } from './RealtimeOverview.messages';

interface Selection {
  serviceIds: string[];
  serviceType?: ServiceType;
}

// resolveSelection reduces the services named in the URL to the ones that share
// a technology. One view of live queries shows one technology - the picker
// enforces it - but a URL can still name both, because starting sessions is not
// restricted and the selection screen hands over everything it started. The
// first service that maps to a running session decides which technology wins.
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
      // Services with no running session are kept: they are not evidence of a
      // mixed selection, and dropping them would silently change the request.
      (serviceId) =>
        (typeByServiceId.get(serviceId) ?? serviceType) === serviceType
    ),
    serviceType,
  };
};

// sessionErrorsMessage explains an empty view whose sessions failed to start, in
// the words the agent reported, instead of letting it read as an idle server.
// Undefined when no selected session failed with a reason.
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
