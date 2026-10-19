import type { DashboardSearchResult } from 'hubs/api/grafanaDashboards';

export type MongoBackupMode = 'snapshot' | 'incremental' | 'pitr';
export type MongoBackupStatus =
  | 'success'
  | 'error'
  | 'inProgress'
  | 'pending'
  | 'other';

export interface MongoSnapshot {
  fetchedAt: number;
  /** Null when the source failed; HubLiveResult.sources says why. */
  services: MongoServiceInput[] | null;
  metrics: MongoMetrics | null;
  alerts: MongoAlertInput[] | null;
  /** Null also while the source has not answered yet. */
  backups: MongoBackupsInput | null;
  qan: { current: QanRowInput[]; baseline: QanRowInput[] } | null;
  realtime: { sessionServiceIds: string[]; queries: RtaQueryInput[] } | null;
  /** Null when Operations is off or failed. */
  operations: MongoOperationsInput | null;
  /** Null also while the catalog has not answered yet. */
  dashboards: {
    catalog: DashboardHit[];
    recent: Array<{ uid: string; visitedAt?: string }>;
  } | null;
}

export interface MongoServiceInput {
  serviceId: string;
  serviceName: string;
  nodeName: string;
  environment?: string;
  cluster?: string;
  replicationSet?: string;
  /** From the management API's `status`. */
  inventoryStatus: 'up' | 'down' | 'unknown';
}

export interface MongoMetrics {
  /** By service_id; a missing key means no series. */
  up: Record<string, { value: 0 | 1; clusterRole: 'mongod' | 'mongos' }>;
  /** Epoch seconds. */
  lastUpAt: Record<string, number>;
  /** member_state is PRIMARY, SECONDARY, ARBITER and so on. */
  self: Record<
    string,
    { rsName: string; clRole?: 'configsvr' | 'shardsvr'; state: string }
  >;
  /** Seconds, secondaries only. */
  lag: Record<string, { now: number; sustained: number }>;
  /** Seconds. */
  oplogWindow: Record<string, number>;
  /** 0..1 */
  oplogFill: Record<string, number>;
  shardsByCluster: Record<string, number>;
  pbmConfiguredByCluster: Record<string, boolean>;
}

export interface MongoAlertInput {
  /** alertname */
  name: string;
  /** Label template_name, such as pmm_mongodb_down. */
  templateName?: string;
  severity?: string;
  serviceName?: string;
  cluster?: string;
  environment?: string;
  /** Label rs_nm or set, for replica set alerts. */
  replicaSet?: string;
  activeAt: string;
  summary?: string;
}

export interface MongoBackupsInput {
  /** settings.backupManagementEnabled */
  enabled: boolean;
  artifacts: Array<{
    artifactId: string;
    name: string;
    serviceId: string;
    serviceName: string;
    mode: MongoBackupMode;
    status: MongoBackupStatus;
    createdAt: string;
  }>;
  scheduled: Array<{
    scheduledBackupId: string;
    name: string;
    serviceId: string;
    serviceName: string;
    mode: MongoBackupMode;
    enabled: boolean;
    cronExpression: string;
    nextRun?: string;
    lastRun?: string;
  }>;
}

/** avgQueryTime in seconds. */
export interface QanRowInput {
  queryId: string;
  fingerprint: string;
  numQueries: number;
  avgQueryTime: number;
}

export interface RtaQueryInput {
  serviceId: string;
  serviceName: string;
  queryText: string;
  operation?: string;
  namespace?: string;
  durationMs: number;
}

export interface MongoOperationsInput {
  /** `${env ?? ''}/${cluster}` to OmCluster.id */
  clusterIds: Record<string, string>;
  summary: { services: number; up: number; lastScanAt?: string };
  failedRuns: Array<{
    id: string;
    kind: 'install' | 'scan';
    target: string;
    finishedAt: string;
    failingNodes?: number;
  }>;
}

export type DashboardHit = DashboardSearchResult;
