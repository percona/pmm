import { RealtimeSessionStatus } from 'types/rta.types';

export interface SessionRow {
  // serviceId or clusterName
  sessionId: string;
  // serviceName or clusterName
  sessionName: string;
  type: 'service' | 'cluster';
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
