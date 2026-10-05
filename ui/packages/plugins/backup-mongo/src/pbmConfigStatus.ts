/**
 * Read a cluster's PBM state from PMM's metrics.
 *
 * What the Configuration tab should answer first is "what is this cluster's backup
 * state right now", and PMM already knows most of it. None of this needs a dispatch,
 * a credential, or SEP.
 *
 * What it cannot answer is the *config document* -- storage type, bucket, region,
 * retention, compression, priorities. There is no `mongodb_pbm_config_*` metric at
 * all, and PBM's own `pbm config -o json` lives behind a payload that runs on the
 * database host. Preloading the form from the live config is a separate, larger
 * piece of work (OM-2); this is deliberately the half that is free.
 */

/** One backup PBM has recorded, as the metrics describe it. */
export interface PbmBackup {
  /** PBM's backup id, which is its start timestamp. */
  name: string;
  /** `logical`, `physical`, `incremental`. */
  type: string;
  /** PBM's own word for it: `done`, `error`, `running`. */
  status: string;
  sizeBytes?: number;
  durationSeconds?: number;
  /** When the backup last changed state, as epoch seconds. */
  lastTransition?: number;
}

/** A cluster's PBM configuration state, as far as metrics can describe it. */
export interface PbmConfigStatus {
  /** Whether PBM has a storage configured. Undefined when the metric is absent. */
  storageConfigured?: boolean;
  /** Whether point-in-time recovery is on. */
  pitrEnabled?: boolean;
  /** Recent backups, newest first. */
  backups: PbmBackup[];
}

interface Series {
  metric: Record<string, string>;
  value: [number, string];
}

const QUERIES = {
  configured: 'mongodb_pbm_cluster_backup_configured',
  pitr: 'mongodb_pbm_cluster_pitr_backup_enabled',
  size: 'mongodb_pbm_backup_size_bytes',
  duration: 'mongodb_pbm_backup_duration_seconds',
  transition: 'mongodb_pbm_backup_last_transition_ts',
} as const;

/** Read a cluster-scoped gauge, which PMM reports once per member. */
function clusterFlag(series: Series[], cluster: string): boolean | undefined {
  const hit = series.find((s) => s.metric.cluster === cluster);
  return hit ? hit.value[1] === '1' : undefined;
}

/**
 * Fold the three per-backup gauges into one row each.
 *
 * They are separate series sharing `name`, `type` and `status` labels, so the
 * backup id is the join key. A backup that reports only some of them still gets a
 * row: a running backup has no size yet, and omitting it would hide the thing most
 * worth seeing.
 */
export function toBackups(
  size: Series[],
  duration: Series[],
  transition: Series[],
  cluster: string
): PbmBackup[] {
  const rows = new Map<string, PbmBackup>();
  const upsert = (s: Series, apply: (row: PbmBackup, v: number) => void) => {
    if (s.metric.cluster !== cluster) {
      return;
    }
    const name = s.metric.name;
    if (!name) {
      return;
    }
    const row = rows.get(name) ?? {
      name,
      type: s.metric.type ?? '',
      status: s.metric.status ?? '',
      // A later series may carry a status the first did not; keep the first non-empty.
    };
    if (!row.status && s.metric.status) {
      row.status = s.metric.status;
    }
    if (!row.type && s.metric.type) {
      row.type = s.metric.type;
    }
    apply(row, Number(s.value[1]));
    rows.set(name, row);
  };

  size.forEach((s) => upsert(s, (r, v) => (r.sizeBytes = v)));
  duration.forEach((s) => upsert(s, (r, v) => (r.durationSeconds = v)));
  transition.forEach((s) => upsert(s, (r, v) => (r.lastTransition = v)));

  // Newest first. PBM names a backup by its start time, so the id sorts correctly
  // even for rows that reported no transition timestamp.
  return [...rows.values()].sort((a, b) => b.name.localeCompare(a.name));
}

async function instant(query: string): Promise<Series[]> {
  const res = await fetch(
    `/prometheus/api/v1/query?query=${encodeURIComponent(query)}`,
    { credentials: 'same-origin', headers: { Accept: 'application/json' } }
  );
  if (!res.ok) {
    throw new Error(`PMM returned ${res.status} for ${query}`);
  }
  return (await res.json())?.data?.result ?? [];
}

/**
 * Fetch one cluster's PBM state.
 *
 * @param cluster The cluster name, as the `cluster` metric label carries it.
 */
export async function fetchPbmConfigStatus(
  cluster: string
): Promise<PbmConfigStatus> {
  // All five together: they are independent gauges and serialising them would
  // multiply the latency of a panel whose whole point is being instant.
  const [configured, pitr, size, duration, transition] = await Promise.all(
    Object.values(QUERIES).map((q) => instant(q).catch(() => [] as Series[]))
  );
  return {
    storageConfigured: clusterFlag(configured, cluster),
    pitrEnabled: clusterFlag(pitr, cluster),
    backups: toBackups(size, duration, transition, cluster),
  };
}
