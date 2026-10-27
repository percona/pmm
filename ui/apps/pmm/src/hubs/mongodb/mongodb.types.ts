import type { DashboardSearchResult } from 'hubs/api/grafanaDashboards';
import type {
  HubNeedsAttentionItem,
  HubRecommendation,
} from 'hubs/template/hub.types';

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
  /** What the alert rules cover; null when alerting is off, failed or has not answered. */
  alertRules: MongoAlertRulesInput | null;
  /** Null also while the source has not answered yet. */
  backups: MongoBackupsInput | null;
  qan: { current: QanRowInput[]; baseline: QanRowInput[] } | null;
  realtime: { sessionServiceIds: string[]; queries: RtaQueryInput[] } | null;
  /** Null when Operations is off or failed. */
  operations: MongoOperationsInput | null;
  /** PMM Extensions' version check by cluster key; null while nothing checks, which is so until P0-1c ships (sample data only). */
  updates: Record<string, MongoUpdateCheck> | null;
  /** Null also while the catalog has not answered yet. */
  dashboards: {
    catalog: DashboardHit[];
    recent: Array<{ uid: string; visitedAt?: string }>;
  } | null;
  /** Sample data only: the Needs attention rows as written for the scenario, which live data derives instead. */
  attention?: HubNeedsAttentionItem[];
  /** Sample data only: the Recommendations rows as written for the scenario. */
  recommendations?: HubRecommendation[];
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
  /** Set and role seen in the last day, for placing a member that reports nothing now. */
  roles: Record<string, { rsName: string; clRole?: 'configsvr' | 'shardsvr' }>;
  /** Epoch seconds the member's current state was first seen within the last hour. */
  stateSince: Record<string, number>;
  /** The server version each member reports, such as 7.0.24-13. */
  versions: Record<string, string>;
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
  /** Label node_name: an alert about the host a member runs on, such as high CPU. */
  nodeName?: string;
  cluster?: string;
  environment?: string;
  /** Label rs_nm or set, for replica set alerts. */
  replicaSet?: string;
  activeAt: string;
  summary?: string;
}

export interface MongoAlertRulesInput {
  /** template_name of every alert rule, MongoDB or not. */
  templates: string[];
  /** Rules made without a template whose query reads a MongoDB metric. */
  custom: number;
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
  /** Upgrades running now; sample data only until Operations reports them. */
  upgrades: MongoUpgradeInput[];
  failedRuns: Array<{
    id: string;
    kind: 'install' | 'scan';
    target: string;
    finishedAt: string;
    failingNodes?: number;
  }>;
}

/** One cluster's version check: the newest patch release in its major.minor series. */
export interface MongoUpdateCheck {
  /** When the check last ran. */
  checkedAt: string;
  /** Absent when the check failed. */
  latest?: string;
}

export interface MongoUpgradeInput {
  /** `${env ?? ''}/${cluster}`, as clusterIds keys it. */
  clusterKey: string;
  /** Members upgraded so far; the next one is in progress. */
  done: number;
  total: number;
  startedAt: string;
  /** The version it upgrades to, such as 7.0.30-16. */
  to?: string;
}

export type DashboardHit = DashboardSearchResult;
