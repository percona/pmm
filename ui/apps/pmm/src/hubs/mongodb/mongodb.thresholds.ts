// PMM-15299's health model, so the hub and Operations' Fleet name a cluster's health alike.
export const MONGODB_HUB_THRESHOLDS = {
  /** Sustained: the lowest value over the last 2 minutes is at or above it. */
  lagWarningSeconds: 10,
  /** A primary oplog window below it degrades the set. */
  oplogWarningSeconds: 24 * 3600,
  /** A filling oplog's window is only as old as the set, so a window counts as short from this fill on. */
  oplogFullRatio: 0.9,
  /** A member without a sample for this long reads No data. */
  staleSeconds: 5 * 60,
  /** How long a member may stay in STARTUP, STARTUP2, RECOVERING or UNKNOWN before it degrades its set. */
  transitionGraceSeconds: 10 * 60,
  /** A version check this old reads as failed, never as up to date; the check runs daily. */
  updateCheckStaleSeconds: 2 * 86400,
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
