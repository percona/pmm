export type BackupStatus =
  | 'BACKUP_STATUS_UNSPECIFIED'
  | 'BACKUP_STATUS_PENDING'
  | 'BACKUP_STATUS_IN_PROGRESS'
  | 'BACKUP_STATUS_PAUSED'
  | 'BACKUP_STATUS_SUCCESS'
  | 'BACKUP_STATUS_ERROR'
  | 'BACKUP_STATUS_DELETING'
  | 'BACKUP_STATUS_FAILED_TO_DELETE'
  | 'BACKUP_STATUS_CLEANUP_IN_PROGRESS';

export type BackupMode =
  | 'BACKUP_MODE_UNSPECIFIED'
  | 'BACKUP_MODE_SNAPSHOT'
  | 'BACKUP_MODE_INCREMENTAL'
  | 'BACKUP_MODE_PITR';

/** `backup.v1.Artifact`, camelCased by the API client. */
export interface BackupArtifact {
  artifactId: string;
  name: string;
  vendor: string;
  locationId?: string;
  locationName?: string;
  serviceId: string;
  serviceName: string;
  dataModel?: string;
  status: BackupStatus;
  createdAt: string;
  mode: BackupMode;
  isShardedCluster?: boolean;
  folder?: string;
}

/** `backup.v1.ScheduledBackup`, camelCased by the API client. */
export interface ScheduledBackup {
  scheduledBackupId: string;
  serviceId: string;
  serviceName: string;
  locationId?: string;
  locationName?: string;
  folder?: string;
  cronExpression: string;
  startTime?: string;
  name: string;
  description?: string;
  enabled: boolean;
  retries?: number;
  retryInterval?: string;
  dataModel?: string;
  mode: BackupMode;
  vendor: string;
  lastRun?: string;
  nextRun?: string;
  retention?: number;
}

export interface ListArtifactsResponse {
  artifacts?: BackupArtifact[];
}

export interface ListScheduledBackupsResponse {
  scheduledBackups?: ScheduledBackup[];
}
