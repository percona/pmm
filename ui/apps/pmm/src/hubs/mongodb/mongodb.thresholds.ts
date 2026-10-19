// Error levels match PMM's own alert template defaults, so the hub and Alerts agree out of the box.
export const MONGODB_HUB_THRESHOLDS = {
  /** Sustained: the lowest value over the last 2 minutes is above it. */
  lagWarningSeconds: 30,
  /** pmm_mongodb_replication_lag default. */
  lagErrorSeconds: 600,
  /** Or lag reaches half the primary's oplog window. */
  lagErrorOplogRatio: 0.5,
  /** pmm_mongodb_oplog_window default. */
  oplogWarningSeconds: 24 * 3600,
  oplogErrorSeconds: 3600,
  /** A window counts as short only once the oplog is this full. */
  oplogFullRatio: 0.9,
  noDataAfterSeconds: 60,
  /** pmm_mongodb_pbm_backup_stale default; a row marker only. */
  backupStaleSeconds: 86400,
  querySlowerRatio: 3,
  /** Calls in the last hour. */
  querySlowerMinCalls: 50,
  querySlowerMinAvgSeconds: 0.1,
  queryRunningLongSeconds: 60,
  operationsLookbackHours: 24,
  backupFailedLookbackHours: 24,
  /** Rows per Queries list. */
  queryRowsMax: 5,
} as const;
