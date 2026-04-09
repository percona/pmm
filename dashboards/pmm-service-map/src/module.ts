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
        name: 'Min edge RPS',
        description: 'Hide edges below this RPS threshold',
        defaultValue: DEFAULT_OPTIONS.minEdgeWeight,
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
      });
  }
);
