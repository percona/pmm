import { api } from 'api/api';
import type {
  BackupArtifact,
  ListArtifactsResponse,
  ListScheduledBackupsResponse,
  ScheduledBackup,
} from './backups.types';

// The hub shows a failed source inside its own section, so the global error toast stays quiet.
const QUIET = { disableNotifications: true };

/** `GET /v1/backups/artifacts`: every backup PMM's backup management made, all vendors. */
export const listArtifacts = async (): Promise<BackupArtifact[]> => {
  const res = await api.get<ListArtifactsResponse>('/backups/artifacts', QUIET);

  return res.data.artifacts ?? [];
};

/** `GET /v1/backups/scheduled`: every backup schedule, all vendors. */
export const listScheduled = async (): Promise<ScheduledBackup[]> => {
  const res = await api.get<ListScheduledBackupsResponse>(
    '/backups/scheduled',
    QUIET
  );

  return res.data.scheduledBackups ?? [];
};
