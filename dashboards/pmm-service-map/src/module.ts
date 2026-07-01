import { PanelPlugin } from '@grafana/data';
import { ServiceMapPanel } from './components/ServiceMapPanel';
import { DEFAULT_OPTIONS, ServiceMapOptions } from './types';

export const plugin = new PanelPlugin<ServiceMapOptions>(ServiceMapPanel).setPanelOptions(
  (builder) => {
    builder
      .addTextInput({
        path: 'promDatasource',
        name: 'Prometheus / VictoriaMetrics datasource',
        description: 'Datasource name for recording-rule metrics (rr_connection_*)',
        defaultValue: DEFAULT_OPTIONS.promDatasource,
      })
      .addTextInput({
        path: 'clickhouseDatasource',
        name: 'ClickHouse datasource',
        description: 'Datasource name for OTLP traces (otel.otel_traces)',
        defaultValue: DEFAULT_OPTIONS.clickhouseDatasource,
      })
      .addNumberInput({
        path: 'errorAmberThreshold',
        name: 'Amber threshold (%)',
        description: 'Error percentage above which an edge turns amber',
        defaultValue: DEFAULT_OPTIONS.errorAmberThreshold,
        settings: { min: 0, max: 100, step: 0.5 },
      })
      .addNumberInput({
        path: 'errorRedThreshold',
        name: 'Red threshold (%)',
        description: 'Error percentage above which an edge turns red',
        defaultValue: DEFAULT_OPTIONS.errorRedThreshold,
        settings: { min: 0, max: 100, step: 0.5 },
      })
      .addNumberInput({
        path: 'minEdgeWeight',
        name: 'Min edge RPS (no TCP bytes)',
        description:
          'Hide edges with RPS below this value only when both TCP byte rates are zero. Does not remove TCP-only edges.',
        defaultValue: DEFAULT_OPTIONS.minEdgeWeight,
        settings: { min: 0, step: 0.1 },
      })
      .addBooleanSwitch({
        path: 'groupByPod',
        name: 'Group by pod (default)',
        description:
          'Initial value for the on-panel View → Group by pod toggle. The toggle on the service map overrides this for the current session.',
        defaultValue: DEFAULT_OPTIONS.groupByPod,
      })
      .addBooleanSwitch({
        path: 'hideWeakEdges',
        name: 'Hide weak edges (default)',
        description:
          'Initial value for the View → Hide weak edges toggle. Toggle on the map overrides for the current session. Uses Weak edge max RPS from options.',
        defaultValue: DEFAULT_OPTIONS.hideWeakEdges,
      })
      .addNumberInput({
        path: 'weakEdgeMaxRps',
        name: 'Weak edge max RPS',
        description:
          'With “Hide weak healthy edges”: minimum L7 req/s to keep a green edge. Edges with rps=0 (TCP-only) are never hidden by this.',
        defaultValue: DEFAULT_OPTIONS.weakEdgeMaxRps,
        settings: { min: 0, step: 0.1 },
      })
      .addSelect({
        path: 'labelMode',
        name: 'Label mode',
        description: 'How service names are displayed on nodes',
        defaultValue: DEFAULT_OPTIONS.labelMode,
        settings: {
          options: [
            { value: 'name', label: 'Service name' },
            { value: 'namespace-name', label: 'Namespace / Service' },
            { value: 'raw', label: 'Raw ID' },
          ],
        },
      })
      .addTextInput({
        path: 'namespaceRenameMap',
        name: 'Namespace rename (JSON)',
        description: 'Optional JSON map of namespace to friendly name, e.g. {"demo":"Application"}',
        defaultValue: '',
      })
      .addTextInput({
        path: 'tracesDashboardUid',
        name: 'Traces dashboard UID',
        description: 'Grafana dashboard UID for ClickHouse OTLP traces (trace ID links use /d/<uid>?var-trace_id=...)',
        defaultValue: DEFAULT_OPTIONS.tracesDashboardUid,
      })
      .addNumberInput({
        path: 'tracesViewPanel',
        name: 'Traces dashboard panel id',
        description: 'viewPanel query parameter when opening a trace from the table',
        defaultValue: DEFAULT_OPTIONS.tracesViewPanel,
        settings: { min: 0, max: 999, step: 1 },
      })
      .addTextInput({
        path: 'kubernetesApiClusterIPs',
        name: 'Kubernetes API ClusterIPs',
        description:
          'Comma-separated IPs of kubernetes.default Service (kubectl get svc kubernetes -n default). Used to label destinations as "Kubernetes API".',
        defaultValue: DEFAULT_OPTIONS.kubernetesApiClusterIPs,
      })
      .addTextInput({
        path: 'kubernetesApiserverEndpointIPs',
        name: 'Kube-apiserver endpoint IPs (optional)',
        description:
          'Comma-separated IPs from kubectl get endpoints kubernetes -n default — labeled "Kubernetes API (control plane)".',
        defaultValue: DEFAULT_OPTIONS.kubernetesApiserverEndpointIPs,
      })
      .addTextInput({
        path: 'destinationLabelOverrides',
        name: 'Destination label overrides (JSON)',
        description:
          'Optional map of exact destination string to label, e.g. {"34.120.177.193:443":"Public API"}. Overrides built-in rules.',
        defaultValue: '',
      })
      .addTextInput({
        path: 'clusterTcpPorts',
        name: 'TCP port filter (optional)',
        description:
          'Comma-separated ports: when set, only TCP byte/failed metrics to those destination ports are drawn (L7 unchanged). Example for Galera/ PXC: 4567,4568,4444. Empty = all TCP.',
        defaultValue: '',
      });
  }
);
