import { NodeThreshold } from 'types/alerting.types';

// A table row is a NodeThreshold plus a stable composite id (`ruleId:paramName`)
// used as the react-hook-form field name — a rule can expose several
// overridable params (e.g. CPU + memory), so the row is per (rule, param).
export interface AlertThresholdRow extends NodeThreshold {
  id: string;
}

// Form values: composite row id -> override value (string while editing).
export type AlertThresholdsFormValues = Record<string, number | undefined>;

export const thresholdRowId = (t: NodeThreshold): string =>
  `${t.ruleId}:${t.paramName}`;
