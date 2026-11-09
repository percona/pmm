import type { Threshold } from 'types/alerting.types';

// A table row is a Threshold plus a stable composite id used as the react-hook-form
// field name, and the rule's title.
//
// The title is not part of the thresholds response: rule metadata churns on rename,
// so Grafana stays authoritative for it and the UI joins it on the `pmm_rule_id`
// label. Rows whose rule has since been deleted keep an empty title rather than
// disappearing.
export interface AlertThresholdRow extends Threshold {
  id: string;
  ruleTitle: string;
}

export interface AlertThresholdsNode {
  nodeId: string;
  nodeName: string;
}

export interface AlertThresholdsProps {
  node: AlertThresholdsNode | null;
  onClose: () => void;
}

// Form values: composite row id -> override value (string while editing).
export type AlertThresholdsFormValues = Record<string, number | undefined>;
