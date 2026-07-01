export interface AlertThresholdRow {
  ruleUid: string;
  alertRuleName: string;
  defaultThreshold?: number;
  overrideThreshold?: number;
  unit?: string;
  supportsOverride?: boolean;
}

// rule uid + override threshold value
export type AlertThresholdsFormValues = Record<string, number | undefined>;
