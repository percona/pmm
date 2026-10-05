/**
 * Read PBM agent topology straight from PMM's metrics.
 *
 * PMM's mongodb_exporter already scrapes PBM state, so which clusters have live
 * agents -- and which member is primary -- is a query, not a background job. The
 * alternative is SEP's inventory sweep, which dispatches a probe onto each database
 * host through Nomad and measured 40-54s per host: a remote allocation only becomes
 * a terminal task when a 30-second reconciliation tick notices it, and no setting
 * below that tick can make it faster.
 *
 * This is same-origin and authenticates with the session cookie the app already
 * holds, so it needs no API key, no proxy and no backend of our own.
 */

/** One member of a replica set, as PBM sees it. */
export interface PbmMember {
  /** PMM's service UUID -- the stable key across re-registrations. */
  serviceId: string;
  /** The service name PMM registered, which is what an operator recognises. */
  serviceName: string;
  /** The replica set this member belongs to. */
  replicaSet: string;
  /** `host:port` as PBM itself names the node, which need not be how PMM reached it. */
  pbmHost: string;
  /** `P` or `S`, straight from the metric label. */
  role: string;
  /** Whether PBM considers this member's agent healthy. */
  healthy: boolean;
  /** Whether this member belongs to a sharded cluster's config replica set. */
  configServer: boolean;
}

/**
 * One deployment an operator configures as a unit.
 *
 * Keyed on the `cluster` label rather than `replica_set`, because PBM's configuration
 * is per *deployment*: a sharded cluster is one thing to configure, not one per shard.
 * A plain replica set has a single entry in `replicaSets`; a sharded cluster has its
 * config set and every shard.
 */
export interface PbmCluster {
  name: string;
  members: PbmMember[];
  /** Replica-set names within this cluster, ordered, config set first when present. */
  replicaSets: string[];
  /** True when the cluster spans more than one replica set. */
  sharded: boolean;
  /** True when every member's agent is healthy; what "can I back this up" turns on. */
  allHealthy: boolean;
}

/**
 * One series per MongoDB service.
 *
 * Without `self="1"` every exporter also reports the peers it can see, so a
 * three-member set answers nine times and six of those describe a member through
 * another node's eyes -- same `host`, different `service_id`.
 */
const PBM_AGENT_QUERY = 'mongodb_pbm_agent_status{self="1"}';

/**
 * Identifies a sharded cluster's config replica set.
 *
 * A metric rather than a name heuristic. Replica sets are named by whoever built the
 * cluster, so matching `-cfg` would be guessing; `mongodb_configsvr` is reported by
 * the server itself and is present only on config members.
 */
const CONFIGSVR_QUERY = 'mongodb_configsvr';

/**
 * The value `mongodb_pbm_agent_status` takes when the agent is healthy.
 *
 * Established by experiment, not documentation -- the metric's help text says only
 * "PBM Agent Status". Stopping one secondary's agent moved that member alone from 0
 * to 2, and restarting it moved it back. 1 has never been observed, so anything
 * non-zero is treated as unhealthy rather than enumerating unseen states.
 */
const AGENT_STATUS_OK = '0';

interface InstantQuerySeries {
  metric: Record<string, string>;
  value: [number, string];
}

/**
 * Shape the instant-query responses into clusters, ordered by name.
 *
 * @param series The `mongodb_pbm_agent_status` result.
 * @param configSeries The `mongodb_configsvr` result, naming config members.
 */
export function toPbmClusters(
  series: InstantQuerySeries[],
  configSeries: InstantQuerySeries[] = []
): PbmCluster[] {
  const configServices = new Set(
    configSeries
      .filter((entry) => entry.value?.[1] === '1')
      .map((entry) => entry.metric?.service_name)
      .filter(Boolean)
  );

  const byCluster = new Map<string, PbmMember[]>();
  for (const entry of series) {
    const labels = entry.metric ?? {};
    const replicaSet = labels.replica_set || labels.replication_set || '';
    // Fall back to the replica set when `cluster` is unset: an unclustered replica
    // set is still one deployment, and dropping it would hide a backup target.
    const clusterName = labels.cluster || replicaSet;
    const serviceId = labels.service_id;
    if (!clusterName || !serviceId) {
      continue;
    }

    const members = byCluster.get(clusterName) ?? [];
    members.push({
      serviceId,
      serviceName: labels.service_name ?? serviceId,
      replicaSet,
      pbmHost: labels.host ?? '',
      role: labels.role ?? '',
      healthy: entry.value?.[1] === AGENT_STATUS_OK,
      configServer: configServices.has(labels.service_name),
    });
    byCluster.set(clusterName, members);
  }

  return [...byCluster.entries()]
    .map(([name, members]) => {
      const ordered = [...members].sort(
        (a, b) =>
          // Config set first, then by replica set, then primary before secondaries.
          Number(b.configServer) - Number(a.configServer) ||
          a.replicaSet.localeCompare(b.replicaSet) ||
          Number(b.role === 'P') - Number(a.role === 'P') ||
          a.serviceName.localeCompare(b.serviceName)
      );
      const replicaSets = [...new Set(ordered.map((m) => m.replicaSet))].filter(
        Boolean
      );
      return {
        name,
        members: ordered,
        replicaSets,
        sharded: replicaSets.length > 1,
        allHealthy: ordered.every((member) => member.healthy),
      };
    })
    .sort((a, b) => a.name.localeCompare(b.name));
}

/**
 * Choose the member that should run `pbm` for this cluster.
 *
 * Secondaries only. A backup read from the primary competes with the workload the
 * cluster exists to serve, and PBM is happy to be driven from any member that has a
 * live agent.
 *
 * On a sharded cluster the config replica set is preferred: PBM's own metadata lives
 * there, so it is the closest thing to where the coordination happens. A shard
 * secondary is the fallback when no config secondary is available.
 *
 * Falls back to a healthy primary only when the cluster has no healthy secondary at
 * all -- a single-node set is a cluster of one, and refusing it would make the common
 * development topology unusable.
 *
 * @param cluster The cluster to pick within.
 * @returns The chosen member, or `undefined` when no member has a live agent.
 */
export function pickExecutor(cluster: PbmCluster): PbmMember | undefined {
  const live = cluster.members.filter((m) => m.healthy);
  const rank = (m: PbmMember) =>
    // Lower sorts first: config secondary, shard secondary, then any primary.
    (m.role === 'P' ? 10 : 0) + (m.configServer ? 0 : 1);
  const ordered = [...live].sort(
    (a, b) => rank(a) - rank(b) || a.serviceName.localeCompare(b.serviceName)
  );
  return ordered[0];
}

async function instantQuery(query: string): Promise<InstantQuerySeries[]> {
  const response = await fetch(
    `/prometheus/api/v1/query?query=${encodeURIComponent(query)}`,
    { credentials: 'same-origin', headers: { Accept: 'application/json' } }
  );
  if (!response.ok) {
    throw new Error(`PMM returned ${response.status} for ${query}`);
  }
  const body = await response.json();
  return body?.data?.result ?? [];
}

/**
 * Fetch PBM topology from PMM.
 *
 * @returns Every deployment PMM is scraping PBM for, ordered by name.
 */
export async function fetchPbmClusters(): Promise<PbmCluster[]> {
  // Both in flight together: the config-server lookup only annotates members, so
  // waiting for it serially would double the latency for no benefit.
  const [agents, configs] = await Promise.all([
    instantQuery(PBM_AGENT_QUERY),
    // A cluster with no config servers is the normal case, so a failure here must
    // not lose the topology -- it only costs the config-server preference.
    instantQuery(CONFIGSVR_QUERY).catch(() => [] as InstantQuerySeries[]),
  ]);
  return toPbmClusters(agents, configs);
}
