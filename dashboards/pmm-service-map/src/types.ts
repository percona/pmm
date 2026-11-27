export interface ServiceMapOptions {
  promDatasource: string;
  clickhouseDatasource: string;
  errorAmberThreshold: number;
  errorRedThreshold: number;
  /**
   * Drop edges with RPS below this when they carry no TCP byte rates (bytes in/out both zero).
   * TCP-only traffic uses bytes and is not removed by this alone.
   */
  minEdgeWeight: number;
  /** Collapse /k8s/ns/pod/container nodes to /k8s/ns/pod */
  groupByPod?: boolean;
  /** Hide healthy low-RPS HTTP edges (see weakEdgeMaxRps); TCP-only edges are kept */
  hideWeakEdges?: boolean;
  /** With hideWeakEdges: minimum L7 req/s to keep a green edge (rps must be > 0) */
  weakEdgeMaxRps?: number;
  labelMode: 'name' | 'namespace-name' | 'raw';
  namespaceRenameMap: Record<string, string>;
  /** Grafana dashboard UID for OTLP traces (ClickHouse) — trace links open this dashboard */
  tracesDashboardUid: string;
  /** Panel id on that dashboard for the trace detail view (used in URL as viewPanel) */
  tracesViewPanel: number;
  /**
   * Comma-separated ClusterIPs of the kubernetes.default Service (443/6443) → label "Kubernetes API".
   * Common: 10.96.0.1, 10.100.0.1, 172.20.0.1
   */
  kubernetesApiClusterIPs: string;
  /**
   * Comma-separated IPs of kube-apiserver endpoints (VPC ENIs behind the Service) → "Kubernetes API (control plane)".
   * Discover with: kubectl get endpoints kubernetes -n default
   */
  kubernetesApiserverEndpointIPs: string;
  /**
   * Optional JSON map of exact destination string → display label (overrides built-in rules).
   * Example: {"34.120.177.193:443":"GCP API"}
   */
  destinationLabelOverrides: string;
  /**
   * Comma-separated destination ports: when non-empty, only TCP metrics (bytes sent/received, failed)
   * whose destination or actual_destination port matches are included. L7 edges are unchanged.
   * Example for Percona XtraDB Cluster / Galera: 4567,4568,4444 (gcomm / IST / SST).
   * Leave empty to include all TCP edges (subject to min edge weight / weak-edge rules).
   */
  clusterTcpPorts?: string;
}

export const DEFAULT_OPTIONS: ServiceMapOptions = {
  promDatasource: '',
  clickhouseDatasource: '',
  errorAmberThreshold: 1,
  errorRedThreshold: 5,
  minEdgeWeight: 0,
  groupByPod: true,
  hideWeakEdges: true,
  weakEdgeMaxRps: 1,
  labelMode: 'name',
  namespaceRenameMap: {},
  tracesDashboardUid: 'otel-traces-clickhouse',
  tracesViewPanel: 20,
  kubernetesApiClusterIPs: '10.96.0.1,10.100.0.1,172.20.0.1',
  kubernetesApiserverEndpointIPs: '',
  destinationLabelOverrides: '',
  clusterTcpPorts: '',
};

export interface ParsedAppId {
  raw: string;
  namespace: string;
  name: string;
  kind: string;
  /** Friendly label for external / IP destinations (does not replace raw id) */
  displayName?: string;
}

export type HealthStatus = 'green' | 'amber' | 'red' | 'unknown';

export interface ServiceNode {
  id: string;
  parsed: ParsedAppId;
  rps: number;
  errPct: number;
  p95Ms: number;
  bytesIn: number;
  bytesOut: number;
  health: HealthStatus;
  /** With groupByPod: containers merged into this pod node */
  podChildContainerCount?: number;
}

export interface ServiceEdge {
  id: string;
  source: string;
  target: string;
  rps: number;
  errPct: number;
  p95Ms: number;
  bytesIn: number;
  bytesOut: number;
  tcpFailed: number;
  health: HealthStatus;
}

export interface ServiceMapData {
  nodes: ServiceNode[];
  edges: ServiceEdge[];
  namespaces: string[];
  /**
   * Every container app_id seen in metric labels for each pod id (includes sidecars with no edges).
   * Used for accurate container counts and trace service name lists when grouped by pod.
   */
  podToContainerAppIds?: Record<string, string[]>;
}

export interface TraceRow {
  timestamp: string;
  traceId: string;
  serviceName: string;
  spanName: string;
  statusCode: string;
  durationMs: number;
}

export type TraceFilter = 'all' | 'errors' | 'slow';

export interface SelectedEdge {
  source: string;
  target: string;
  sourceLabel: string;
  targetLabel: string;
  edge: ServiceEdge;
  /** Resolved app_id for source (always an app_id like /k8s/ns/name) */
  sourceAppId: string;
  /** Resolved app_id for target (may still be IP if unresolved) */
  targetAppId: string;
}

export interface SelectedNode {
  id: string;
  label: string;
  node: ServiceNode;
  outgoingEdges: ServiceEdge[];
  outgoingLabels: string[];
  /** Set when groupByPod and this node is a pod aggregate — OTel ServiceName values */
  traceServiceNames?: string[];
  /** Containers under this pod (from ungrouped data) */
  childContainers?: ServiceNode[];
  /** Pod-level view: same-pod container↔container edges not drawn */
  internalSamePodEdgesHidden?: number;
}
