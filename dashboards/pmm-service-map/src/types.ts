export interface ServiceMapOptions {
  promDatasource: string;
  clickhouseDatasource: string;
  errorAmberThreshold: number;
  errorRedThreshold: number;
  minEdgeWeight: number;
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
}

export const DEFAULT_OPTIONS: ServiceMapOptions = {
  promDatasource: '',
  clickhouseDatasource: '',
  errorAmberThreshold: 1,
  errorRedThreshold: 5,
  minEdgeWeight: 0,
  labelMode: 'name',
  namespaceRenameMap: {},
  tracesDashboardUid: 'otel-traces-clickhouse',
  tracesViewPanel: 20,
  kubernetesApiClusterIPs: '10.96.0.1,10.100.0.1,172.20.0.1',
  kubernetesApiserverEndpointIPs: '',
  destinationLabelOverrides: '',
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
}
