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
  /** `host:port` as PBM itself names the node, which need not be how PMM reached it. */
  pbmHost: string;
  /** `P` or `S`, straight from the metric label. */
  role: string;
  /** Whether PBM considers this member's agent healthy. */
  healthy: boolean;
}

/** One replica set and the agents PBM reports for it. */
export interface PbmCluster {
  name: string;
  members: PbmMember[];
  /** True when every member's agent is healthy; what "can I back this up" turns on. */
  allHealthy: boolean;
}

/**
 * One series per service.
 *
 * Without `self="1"` every exporter also reports the peers it can see, so a
 * three-member set answers nine times and six of those describe a member through
 * another node's eyes -- same `host`, different `service_id`. The filter is what
 * makes `service_id` mean "this series is about that service".
 */
const PBM_AGENT_QUERY = 'mongodb_pbm_agent_status{self="1"}';

/**
 * The value the metric takes when the agent is healthy.
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

/** Shape the instant-query response into clusters, ordered by name. */
export function toPbmClusters(series: InstantQuerySeries[]): PbmCluster[] {
  const byCluster = new Map<string, PbmMember[]>();

  for (const entry of series) {
    const labels = entry.metric ?? {};
    const clusterName = labels.replica_set || labels.replication_set;
    const serviceId = labels.service_id;
    // A series without a set or a service id cannot be placed or keyed. Skipping it
    // is not an error worth surfacing: it simply is not about a member we can name.
    if (!clusterName || !serviceId) {
      continue;
    }

    const members = byCluster.get(clusterName) ?? [];
    members.push({
      serviceId,
      serviceName: labels.service_name ?? serviceId,
      pbmHost: labels.host ?? '',
      role: labels.role ?? '',
      healthy: entry.value?.[1] === AGENT_STATUS_OK,
    });
    byCluster.set(clusterName, members);
  }

  return [...byCluster.entries()]
    .map(([name, members]) => ({
      name,
      // Primary first, then by name, so the list does not reshuffle on election
      // beyond the one member whose role actually changed.
      members: [...members].sort(
        (a, b) =>
          Number(b.role === 'P') - Number(a.role === 'P') ||
          a.serviceName.localeCompare(b.serviceName)
      ),
      allHealthy: members.every((member) => member.healthy),
    }))
    .sort((a, b) => a.name.localeCompare(b.name));
}

/**
 * Fetch PBM topology from PMM.
 *
 * @returns Every replica set PMM is scraping PBM for, ordered by name.
 */
export async function fetchPbmClusters(): Promise<PbmCluster[]> {
  const response = await fetch(
    `/prometheus/api/v1/query?query=${encodeURIComponent(PBM_AGENT_QUERY)}`,
    { credentials: 'same-origin', headers: { Accept: 'application/json' } }
  );
  if (!response.ok) {
    throw new Error(`PMM returned ${response.status} for the PBM agent query`);
  }
  const body = await response.json();
  return toPbmClusters(body?.data?.result ?? []);
}
