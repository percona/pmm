import { RealtimeSessionStatus } from 'types/rta.types';
import { ServiceType } from 'types/services.types';

export interface SessionRow {
  // serviceId or clusterName
  sessionId: string;
  // serviceName or clusterName
  sessionName: string;
  type: 'service' | 'cluster';
  // For a cluster row this is the technology shared by its services.
  serviceType?: ServiceType;
  startTime: string;
  status: RealtimeSessionStatus;
  serviceSessions: SessionRow[];
}

export type ModalType =
  | 'stop'
  | 'stop-all'
  | 'stop-selected'
  | 'new-session'
  | null;
