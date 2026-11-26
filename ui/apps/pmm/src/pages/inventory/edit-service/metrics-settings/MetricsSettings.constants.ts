import { MetricsMode } from 'types/services.types';
import { Messages } from '../EditService.messages';
import type { SettingName } from '../EditService.types';

const { metrics } = Messages;

export const METRICS_MODE_OPTIONS = [
  { value: MetricsMode.pull, label: metrics.pull },
  { value: MetricsMode.push, label: metrics.push },
];

// Text and number settings, in the order the form shows them
export const FIELDS: [SettingName, string, string, boolean][] = [
  [
    'disableCollectors',
    metrics.disableCollectors,
    metrics.disableCollectorsDescription,
    false,
  ],
  [
    'tablestatsGroupTableLimit',
    metrics.tablestatsGroupTableLimit,
    metrics.tablestatsGroupTableLimitDescription,
    true,
  ],
  [
    'statsCollections',
    metrics.statsCollections,
    metrics.statsCollectionsDescription,
    false,
  ],
  [
    'collectionsLimit',
    metrics.collectionsLimit,
    metrics.collectionsLimitDescription,
    true,
  ],
  [
    'autoDiscoveryLimit',
    metrics.autoDiscoveryLimit,
    metrics.autoDiscoveryLimitDescription,
    true,
  ],
  [
    'maxExporterConnections',
    metrics.maxExporterConnections,
    metrics.maxExporterConnectionsDescription,
    true,
  ],
];

export const SWITCHES: [SettingName, string][] = [
  ['exposeExporter', metrics.exposeExporter],
  ['enableAllCollectors', metrics.enableAllCollectors],
  ['enableDiagnosticDataHistograms', metrics.enableDiagnosticDataHistograms],
];
